package fixture_test

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	netmail "net/mail"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ChiragAgg5k/mailslot/internal/failure"
	"github.com/ChiragAgg5k/mailslot/internal/fixture"
	"github.com/ChiragAgg5k/mailslot/internal/mail"
)

const data = "testdata"

func open(t *testing.T, state string) *fixture.Backend {
	t.Helper()
	t.Setenv(fixture.FaultVariable, "")
	backend, err := fixture.Open(data, state)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	return backend
}

func identifiers(summaries []mail.Summary) []int {
	ids := make([]int, 0, len(summaries))
	for _, summary := range summaries {
		ids = append(ids, summary.ID)
	}
	return ids
}

func same(first, second []int) bool {
	if len(first) != len(second) {
		return false
	}
	for index := range first {
		if first[index] != second[index] {
			return false
		}
	}
	return true
}

func code(t *testing.T, err error, want failure.Code) {
	t.Helper()
	if got := failure.CodeOf(err); got != want {
		t.Fatalf("exit code %d (%v), want %d", got, err, want)
	}
}

func TestAccountsAreRealOnly(t *testing.T) {
	backend := open(t, "")
	accounts, err := backend.Accounts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(accounts) != 2 || accounts[0].Name != "Work" || accounts[1].Name != "Home" {
		t.Fatalf("accounts %+v", accounts)
	}
	status, err := backend.Ping(context.Background())
	if err != nil || status != (mail.Status{Version: fixture.Version, Accounts: 2}) {
		t.Fatalf("ping %+v %v", status, err)
	}
}

func TestMailboxesCountMessages(t *testing.T) {
	backend := open(t, "")
	mailboxes, err := backend.Mailboxes(context.Background(), "work")
	if err != nil {
		t.Fatal(err)
	}
	if len(mailboxes) != 4 || mailboxes[0] != (mail.Mailbox{Name: "Inbox", Unread: 1, Messages: 3}) || mailboxes[1] != (mail.Mailbox{Name: "Archive", Unread: 1, Messages: 1}) {
		t.Fatalf("mailboxes %+v", mailboxes)
	}
	_, err = backend.Mailboxes(context.Background(), "Nowhere")
	code(t, err, failure.NotFound)
}

func TestListNewestFirst(t *testing.T) {
	backend := open(t, "")
	ctx := context.Background()
	inbox := mail.Location{Account: "WORK", Mailbox: "inbox"}
	all, err := backend.List(ctx, inbox, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	if !same(identifiers(all), []int{1003, 1001, 1004}) {
		t.Fatalf("list %v", identifiers(all))
	}
	if all[0].Flagged || all[0].Mailbox != "Inbox" || all[0].MessageID != "cafe@example.com" {
		t.Fatalf("summary %+v", all[0])
	}
	limited, _ := backend.List(ctx, inbox, 2, false)
	unread, _ := backend.List(ctx, inbox, 0, true)
	if !same(identifiers(limited), []int{1003, 1001}) || !same(identifiers(unread), []int{1003}) {
		t.Fatalf("limited %v unread %v", identifiers(limited), identifiers(unread))
	}
	_, err = backend.List(ctx, mail.Location{Account: "Work", Mailbox: "Nowhere"}, 0, false)
	code(t, err, failure.NotFound)
	local, err := backend.List(ctx, mail.Location{Account: mail.Local, Mailbox: "fixture"}, 0, false)
	if err != nil || !same(identifiers(local), []int{1005}) || local[0].Mailbox != "Import/fixture" {
		t.Fatalf("local leaf name %v %v", local, err)
	}
}

func TestSearchAndsConditions(t *testing.T) {
	backend := open(t, "")
	ctx := context.Background()
	inbox := mail.Location{Account: "Work", Mailbox: "Inbox"}
	since := time.Date(2026, 3, 2, 9, 1, 0, 0, time.UTC)
	cases := []struct {
		name  string
		query mail.Query
		want  []int
	}{
		{"text in sender", mail.Query{Location: inbox, Text: "zoe"}, []int{1003}},
		{"text in subject", mail.Query{Location: inbox, Text: "QUARTERLY"}, []int{1001}},
		{"text ignores body", mail.Query{Location: inbox, Text: "thursday"}, []int{}},
		{"text with body", mail.Query{Location: inbox, Text: "thursday", Body: true}, []int{1001}},
		{"since inclusive", mail.Query{Location: inbox, Since: &since}, []int{1003, 1001}},
		{"from and unread", mail.Query{Location: inbox, From: "example.com", Unread: true}, []int{1003}},
		{"subject and from disagree", mail.Query{Location: inbox, Subject: "planning", From: "zoe"}, []int{}},
		{"limit", mail.Query{Location: inbox, Limit: 1}, []int{1003}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			found, err := backend.Search(ctx, test.query)
			if err != nil {
				t.Fatal(err)
			}
			if !same(identifiers(found), test.want) {
				t.Fatalf("found %v, want %v", identifiers(found), test.want)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	backend := open(t, "")
	ctx := context.Background()
	everywhere := mail.Target{Location: mail.Location{Account: "Work"}, Reference: mail.Reference{MessageID: "plan-2@example.com"}}
	summary, err := backend.Resolve(ctx, everywhere)
	if err != nil || summary.ID != 1002 || summary.Mailbox != "Archive" || !summary.Flagged {
		t.Fatalf("resolve by Message-ID %+v %v", summary, err)
	}

	inInbox := everywhere
	inInbox.Mailbox = "Inbox"
	_, err = backend.Resolve(ctx, inInbox)
	code(t, err, failure.NotFound)
	if failure.HintOf(err) == "" || strings.Contains(failure.HintOf(err), "numeric id") {
		t.Fatalf("Message-ID miss hint %q", failure.HintOf(err))
	}

	byID := mail.Target{Location: mail.Location{Account: "Work", Mailbox: "Inbox"}, Reference: mail.Reference{ID: 1002}}
	_, err = backend.Resolve(ctx, byID)
	code(t, err, failure.NotFound)
	if !strings.Contains(failure.HintOf(err), "numeric id") {
		t.Fatalf("id miss hint %q", failure.HintOf(err))
	}

	_, err = backend.Resolve(ctx, mail.Target{Location: mail.Location{Account: "Nobody"}, Reference: mail.Reference{ID: 1001}})
	code(t, err, failure.NotFound)

	message, err := backend.Read(ctx, mail.Target{Location: mail.Location{Account: "Work", Mailbox: "Archive"}, Reference: mail.Reference{ID: 1002}})
	if err != nil || len(message.Attachments) != 2 || message.Attachments[0].Size != 20 || len(message.Cc) != 1 || message.Mailbox != "Archive" {
		t.Fatalf("read %+v %v", message, err)
	}
}

func TestChangesStayInMemoryWithoutState(t *testing.T) {
	backend := open(t, "")
	ctx := context.Background()
	target := mail.Target{Location: mail.Location{Account: "Work", Mailbox: "Inbox"}, Reference: mail.Reference{ID: 1001}}
	if _, err := backend.Mark(ctx, target, false); err != nil {
		t.Fatal(err)
	}
	summary, err := backend.Resolve(ctx, target)
	if err != nil || summary.Read {
		t.Fatalf("mark not kept in memory: %+v %v", summary, err)
	}
	if fresh, _ := open(t, "").Resolve(ctx, target); !fresh.Read {
		t.Fatal("a fresh backend saw another backend's memory")
	}
}

func TestMoveAssignsNewIDAndPersists(t *testing.T) {
	state := filepath.Join(t.TempDir(), "state.json")
	backend := open(t, state)
	ctx := context.Background()
	target := mail.Target{Location: mail.Location{Account: "Work", Mailbox: "Inbox"}, Reference: mail.Reference{ID: 1001}}

	moved, err := backend.Move(ctx, target, "archive")
	if err != nil {
		t.Fatal(err)
	}
	if moved.ID != 1101 || moved.Mailbox != "Archive" || moved.MessageID != "plan-1@example.com" {
		t.Fatalf("moved %+v", moved)
	}
	if _, err := backend.Resolve(ctx, target); failure.CodeOf(err) != failure.NotFound {
		t.Fatalf("old id still resolves: %v", err)
	}
	if _, err := backend.Move(ctx, mail.Target{Location: target.Location, Reference: mail.Reference{ID: 1003}}, "Nowhere"); failure.CodeOf(err) != failure.NotFound {
		t.Fatalf("move to unknown mailbox: %v", err)
	}
	if _, err := backend.Flag(ctx, mail.Target{Location: target.Location, Reference: mail.Reference{ID: 1003}}, mail.Blue); err != nil {
		t.Fatal(err)
	}

	reopened := open(t, state)
	archived, err := reopened.List(ctx, mail.Location{Account: "Work", Mailbox: "Archive"}, 0, false)
	if err != nil || !same(identifiers(archived), []int{1002, 1101}) {
		t.Fatalf("archive after reopen %v %v", identifiers(archived), err)
	}
	flagged, err := reopened.Resolve(ctx, mail.Target{Location: mail.Location{Account: "Work"}, Reference: mail.Reference{MessageID: "cafe@example.com"}})
	if err != nil || !flagged.Flagged {
		t.Fatalf("flag after reopen %+v %v", flagged, err)
	}

	untouched := open(t, "")
	if inbox, _ := untouched.List(ctx, mail.Location{Account: "Work", Mailbox: "Inbox"}, 0, false); len(inbox) != 3 {
		t.Fatalf("fixture directory changed: %v", identifiers(inbox))
	}
	entries, err := os.ReadDir(filepath.Dir(state))
	if err != nil || len(entries) != 1 {
		t.Fatalf("state directory holds %v %v, want only the state file", entries, err)
	}
}

func TestStateMustNotLiveInFixtureDirectory(t *testing.T) {
	t.Setenv(fixture.FaultVariable, "")
	_, err := fixture.Open(data, filepath.Join(data, "state.json"))
	code(t, err, failure.Usage)
}

func TestSourceIsMIME(t *testing.T) {
	backend := open(t, "")
	ctx := context.Background()

	raw, err := backend.Source(ctx, mail.Target{Location: mail.Location{Account: "Work"}, Reference: mail.Reference{MessageID: "plan-2@example.com"}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ReplaceAll(raw, "\r\n", ""), "\n") {
		t.Fatal("source has bare LF line endings")
	}
	parsed, err := netmail.ReadMessage(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{
		"Message-Id":   "<plan-2@example.com>",
		"In-Reply-To":  "<plan-1@example.com>",
		"References":   "<plan-1@example.com>",
		"Mime-Version": "1.0",
		"Date":         "Tue, 03 Mar 2026 09:00:00 +0000",
		"Cc":           `"Robin" <robin@example.com>`,
	} {
		if got := parsed.Header.Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
	kind, parameters, err := mime.ParseMediaType(parsed.Header.Get("Content-Type"))
	if err != nil || kind != "multipart/mixed" {
		t.Fatalf("content type %q %v", kind, err)
	}
	reader := multipart.NewReader(parsed.Body, parameters["boundary"])
	var bodies []string
	var names []string
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		var content io.Reader = part
		if part.Header.Get("Content-Transfer-Encoding") == "base64" {
			content = base64.NewDecoder(base64.StdEncoding, part)
		}
		contents, err := io.ReadAll(content)
		if err != nil {
			t.Fatal(err)
		}
		bodies = append(bodies, string(contents))
		names = append(names, part.FileName())
	}
	if len(bodies) != 3 || !strings.Contains(bodies[0], "INV-77") || bodies[1] != "1. Budget\n2. Hiring\n" || names[2] != "agenda.txt" {
		t.Fatalf("parts %q names %q", bodies, names)
	}

	raw, err = backend.Source(ctx, mail.Target{Location: mail.Location{Account: "Work", Mailbox: "Inbox"}, Reference: mail.Reference{ID: 1003}})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err = netmail.ReadMessage(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	encoded := parsed.Header.Get("Subject")
	decoded, err := new(mime.WordDecoder).DecodeHeader(encoded)
	if err != nil || decoded != "Café in Zürich — 東京 ✈" || encoded == decoded {
		t.Fatalf("subject %q decodes to %q %v", encoded, decoded, err)
	}
	sender, err := parsed.Header.AddressList("From")
	if err != nil || sender[0].Name != "Zoë Müller" {
		t.Fatalf("from %v %v", sender, err)
	}
	body, _ := io.ReadAll(parsed.Body)
	if !strings.Contains(string(body), "Gr=C3=BC=C3=9Fe") {
		t.Fatalf("body not quoted-printable: %q", body)
	}
}

func TestSaveAttachmentsRenamesRepeats(t *testing.T) {
	backend := open(t, "")
	directory := t.TempDir()
	saved, err := backend.SaveAttachments(context.Background(), mail.Target{Location: mail.Location{Account: "Work", Mailbox: "Archive"}, Reference: mail.Reference{ID: 1002}}, directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(saved) != 2 || saved[0].Name != "agenda.txt" || saved[1].Name != "agenda-2.txt" {
		t.Fatalf("saved %+v", saved)
	}
	contents, err := os.ReadFile(filepath.Join(directory, "agenda-2.txt"))
	if err != nil || string(contents) != "second copy\n" {
		t.Fatalf("second copy %q %v", contents, err)
	}
}

func TestFetchDeliversOnce(t *testing.T) {
	state := filepath.Join(t.TempDir(), "state.json")
	backend := open(t, state)
	ctx := context.Background()
	inbox := mail.Location{Account: "Work", Mailbox: "Inbox"}
	before := time.Now().UTC().Truncate(time.Second)

	if err := backend.Fetch(ctx, "Home"); err != nil {
		t.Fatal(err)
	}
	if err := backend.Fetch(ctx, "work"); err != nil {
		t.Fatal(err)
	}
	listed, err := backend.List(ctx, inbox, 0, false)
	if err != nil || len(listed) != 4 || listed[0].ID != 1100 || listed[0].Read || listed[0].Received.Before(before) {
		t.Fatalf("after fetch %+v %v", listed, err)
	}
	if err := open(t, state).Fetch(ctx, "Work"); err != nil {
		t.Fatal(err)
	}
	again, _ := backend.List(ctx, inbox, 0, false)
	if len(again) != 4 {
		t.Fatalf("redelivered: %v", identifiers(again))
	}
}

func TestCompose(t *testing.T) {
	backend := open(t, "")
	ctx := context.Background()

	_, err := backend.Compose(ctx, mail.Draft{Account: "local", To: []string{"x@example.com"}})
	code(t, err, failure.Usage)
	_, err = backend.Compose(ctx, mail.Draft{Account: "Work", From: "alex@example.org"})
	code(t, err, failure.Usage)
	_, err = backend.Compose(ctx, mail.Draft{Account: "Work", Signature: "Missing"})
	code(t, err, failure.NotFound)

	delivery, err := backend.Compose(ctx, mail.Draft{Account: "Work", From: "Alex <A@example.com>", To: []string{"Robin <robin@example.com>"}, Subject: "Hello", Body: "Hi\n", Signature: "work"})
	if err != nil || delivery != (mail.Delivery{Sent: false, Sender: "Alex <A@example.com>"}) {
		t.Fatalf("draft %+v %v", delivery, err)
	}
	drafts, _ := backend.List(ctx, mail.Location{Account: "Work", Mailbox: "Drafts"}, 0, false)
	if len(drafts) != 1 {
		t.Fatalf("drafts %+v", drafts)
	}
	message, _ := backend.Read(ctx, mail.Target{Location: mail.Location{Account: "Work", Mailbox: "Drafts"}, Reference: mail.Reference{ID: drafts[0].ID}})
	if message.Content != "Hi\n\nAlex\nExample Co." || message.To[0].Address != "robin@example.com" {
		t.Fatalf("draft message %+v", message)
	}

	delivery, err = backend.Compose(ctx, mail.Draft{Account: "home", To: []string{"robin@example.com"}, Subject: "Sent", Send: true})
	if err != nil || delivery != (mail.Delivery{Sent: true, Sender: "alex@example.org"}) {
		t.Fatalf("send %+v %v", delivery, err)
	}
	sent, _ := backend.List(ctx, mail.Location{Account: "Home", Mailbox: "Sent Messages"}, 0, false)
	if len(sent) != 1 || sent[0].Subject != "Sent" {
		t.Fatalf("sent %+v", sent)
	}
}

func TestFaults(t *testing.T) {
	cases := []struct {
		fault string
		want  failure.Code
	}{
		{"permission", failure.Permission},
		{"timeout", failure.Busy},
	}
	for _, test := range cases {
		t.Run(test.fault, func(t *testing.T) {
			t.Setenv(fixture.FaultVariable, test.fault)
			backend, err := fixture.Open(data, "")
			if err != nil {
				t.Fatal(err)
			}
			_, err = backend.Accounts(context.Background())
			code(t, err, test.want)
			_, err = backend.Compose(context.Background(), mail.Draft{Account: "Work"})
			code(t, err, test.want)
		})
	}
	t.Setenv(fixture.FaultVariable, "sometimes")
	_, err := fixture.Open(data, "")
	code(t, err, failure.General)
}
