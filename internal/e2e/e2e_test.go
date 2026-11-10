//go:build e2e

// Package e2e_test runs mailslot's commands against the real Apple Mail app.
//
// This suite changes mail: it marks, flags, moves and deletes messages, and on a real account it sends
// a message to the account's own address and replies to it there. It is written for a dedicated test account
// or for the hosted CI runner, whose Mail has only local mailboxes. Never run it on a personal Mac.
//
// It runs only when MAILSLOT_E2E_ACCOUNT names the account to use (local for the On My Mac mailboxes)
// and refuses to start when MAILSLOT_FIXTURE is set. MAILSLOT_E2E_MAILBOX names the mailbox to take
// messages from (default Inbox), MAILSLOT_E2E_TARGET the mailbox to move them to and back from, and
// MAILSLOT_E2E_MARKER a subject token that some message there carries (default mailslot-e2e-marker).
// Each test puts back what it changed.
package e2e_test

import (
	"fmt"
	netmail "net/mail"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ChiragAgg5k/mailslot/internal/failure"
	"github.com/ChiragAgg5k/mailslot/internal/mail"
)

const (
	statusFailed    = "fail"
	statusPassed    = "ok"
	statusWarned    = "warn"
	accountsCheck   = "accounts"
	messageIDHeader = "message-id:"
	flagColor       = "red"
	expectedCode    = "123456"
	codeBody        = "Your code is " + expectedCode + "."
	replyBody       = "mailslot end-to-end reply; safe to delete."
	waitTimeout     = "5m"
	waitInterval    = "5s"
	waitHeadStart   = 10 * time.Second
	subjectPrefix   = "mailslot-e2e-"
	csvName         = "people.csv"
	noArchive       = "the account has no archive mailbox"
	allMail         = "All Mail"
)

// TestReadAndTriage covers reading and triage. It runs for local too, since the CI runner has no accounts.
func TestReadAndTriage(t *testing.T) {
	current := begin(t)

	t.Run("doctor", func(t *testing.T) {
		var found diagnosis
		current.run(t, "doctor").expect(t, failure.Success).decode(t, &found)
		if len(found.Checks) == 0 {
			t.Fatal("doctor reported no checks")
		}
		for _, item := range found.Checks {
			if item.Status == statusFailed {
				t.Errorf("doctor check %s failed: %s", item.Name, item.Detail)
			}
			if item.Name == accountsCheck && item.Status != statusPassed && item.Status != statusWarned {
				t.Errorf("doctor accounts check is %q, want ok, or warn when Mail has no accounts", item.Status)
			}
		}
	})

	t.Run("accounts", func(t *testing.T) {
		var accounts []account
		current.run(t, "accounts").expect(t, failure.Success).decode(t, &accounts)
		for _, item := range accounts {
			if strings.EqualFold(item.Name, current.account) {
				return
			}
		}
		t.Fatalf("accounts lists %d accounts, none named %s", len(accounts), current.account)
	})

	t.Run("mailboxes", func(t *testing.T) {
		var mailboxes []mailbox
		current.run(t, "mailboxes").expect(t, failure.Success).decode(t, &mailboxes)
		for _, item := range mailboxes {
			if item.Name == "" || item.Messages < 0 || item.Unread < 0 {
				t.Errorf("mailbox entry has an empty name or negative counts: %+v", item)
			}
		}
	})

	rows := current.arrived(t)
	t.Logf("source mailbox lists %d messages", len(rows))

	t.Run("list", func(t *testing.T) {
		for _, row := range rows {
			if row.ID <= 0 {
				t.Errorf("listed message has id %d", row.ID)
			}
			if !same(row.Mailbox, current.source) {
				t.Errorf("message %d is listed in %q, want %q", row.ID, row.Mailbox, current.source)
			}
		}
		for position := 1; position < len(rows); position++ {
			earlier, later := rows[position].Received, rows[position-1].Received
			if earlier != nil && (later == nil || later.Before(*earlier)) {
				t.Errorf("message %d is listed before the newer message %d", rows[position-1].ID, rows[position].ID)
			}
		}
		var newest []summary
		current.run(t, "list", "--mailbox", current.source, "--limit", "1").expect(t, failure.Success).decode(t, &newest)
		if len(newest) != 1 {
			t.Fatalf("list --limit 1 returned %d messages", len(newest))
		}
		if first := rows[0].Received; first != nil && (newest[0].Received == nil || newest[0].Received.Before(*first)) {
			t.Errorf("list --limit 1 returned message %d, which is older than the newest listed message %d", newest[0].ID, rows[0].ID)
		}
	})

	t.Run("read", func(t *testing.T) {
		row := identified(t, rows)[0]
		byID := current.read(t, strconv.Itoa(row.ID), current.source)
		if byID.ID != row.ID || byID.MessageID != row.MessageID {
			t.Fatalf("read %d returned id %d, Message-ID match %t", row.ID, byID.ID, byID.MessageID == row.MessageID)
		}
		for _, reference := range []string{row.MessageID, "<" + row.MessageID + ">"} {
			var byMessageID message
			current.run(t, "read", reference).expect(t, failure.Success).decode(t, &byMessageID)
			if byMessageID.MessageID != row.MessageID {
				t.Errorf("read by Message-ID (brackets %t) returned another message", strings.HasPrefix(reference, "<"))
			}
		}
	})

	t.Run("search", func(t *testing.T) {
		marker := valueOr(os.Getenv(markerVariable), defaultMarker)
		var found []summary
		current.run(t, "search", marker, "--mailbox", current.source, "--limit", "0").expect(t, failure.Success).decode(t, &found)
		if len(found) == 0 {
			if continuous() {
				t.Fatalf("search for the marker found nothing in %q", current.source)
			}
			t.Skipf("no message in %q carries the marker; set %s to a token one of them holds", current.source, markerVariable)
		}
		for _, row := range found {
			if !strings.Contains(strings.ToLower(row.Subject+" "+row.Sender), strings.ToLower(marker)) {
				t.Errorf("search returned message %d whose subject and sender lack the marker", row.ID)
			}
		}
		var shouted []summary
		current.run(t, "search", strings.ToUpper(marker), "--mailbox", current.source, "--limit", "0").expect(t, failure.Success).decode(t, &shouted)
		if len(shouted) != len(found) {
			t.Errorf("search for the marker in capitals found %d messages, want %d", len(shouted), len(found))
		}
		t.Logf("search found %d messages", len(found))
	})

	t.Run("attachments", func(t *testing.T) {
		for _, row := range rows {
			full := current.read(t, strconv.Itoa(row.ID), current.source)
			if len(full.Attachments) == 0 {
				continue
			}
			directory := t.TempDir()
			var files []saved
			current.run(t, "attachments", strconv.Itoa(row.ID), "--mailbox", current.source, "--output", directory).expect(t, failure.Success).decode(t, &files)
			if len(files) != len(full.Attachments) {
				t.Fatalf("message %d lists %d attachments but %d were saved", row.ID, len(full.Attachments), len(files))
			}
			for _, file := range files {
				if filepath.Dir(file.Path) != directory {
					t.Errorf("attachment saved outside the output directory")
				}
				if _, err := os.Stat(file.Path); err != nil {
					t.Errorf("saved attachment is missing: %v", err)
				}
			}
			t.Logf("saved %d attachments of message %d", len(files), row.ID)
			return
		}
		if continuous() {
			t.Fatalf("none of the %d messages in %q has attachments", len(rows), current.source)
		}
		t.Skipf("none of the %d messages in %q has attachments", len(rows), current.source)
	})

	t.Run("raw", func(t *testing.T) {
		row := identified(t, rows)[0]
		var printed source
		current.run(t, "read", strconv.Itoa(row.ID), "--mailbox", current.source, "--raw").expect(t, failure.Success).decode(t, &printed)
		if !strings.Contains(strings.ToLower(printed.Source), messageIDHeader) {
			t.Errorf("the raw source of message %d (%d bytes) has no Message-ID header", row.ID, len(printed.Source))
		}
		if printed.ID != row.ID || printed.Path != "" {
			t.Errorf("read --raw reported id %d and path %q, want id %d and no path", printed.ID, printed.Path, row.ID)
		}

		path := filepath.Join(t.TempDir(), "message.eml")
		var written source
		current.run(t, "read", strconv.Itoa(row.ID), "--mailbox", current.source, "--raw", "--output", path).expect(t, failure.Success).decode(t, &written)
		if written.Path != path || written.Source != "" {
			t.Errorf("read --raw --output reported path %q and %d source bytes, want %q and none", written.Path, len(written.Source), path)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read the saved source: %v", err)
		}
		if !strings.Contains(strings.ToLower(string(content)), messageIDHeader) {
			t.Errorf("the saved source (%d bytes) has no Message-ID header", len(content))
		}
	})

	t.Run("mark", func(t *testing.T) {
		row := current.messages(t)[0]
		id := strconv.Itoa(row.ID)
		original := current.read(t, id, current.source).Read
		restore := "--unread"
		if original {
			restore = "--read"
		}
		t.Cleanup(func() { current.run(t, "mark", id, "--mailbox", current.source, restore) })

		for _, state := range []bool{true, false} {
			option := "--unread"
			if state {
				option = "--read"
			}
			reported := current.changes(t, "mark", id, "--mailbox", current.source, option)
			if reported.ID != row.ID || reported.NewID != nil || reported.DryRun {
				t.Fatalf("mark %s reported id %d, newId set %t, dry run %t; want id %d unchanged", option, reported.ID, reported.NewID != nil, reported.DryRun, row.ID)
			}
			after := current.read(t, id, current.source)
			if after.Read != state {
				t.Fatalf("after mark %s message %d has read %t", option, row.ID, after.Read)
			}
		}
	})

	t.Run("flag", func(t *testing.T) {
		var row summary
		for _, candidate := range current.messages(t) {
			if !candidate.Flagged {
				row = candidate
				break
			}
		}
		if row.ID == 0 {
			t.Skipf("every listed message in %q is flagged already, and its colour could not be restored", current.source)
		}
		id := strconv.Itoa(row.ID)
		t.Cleanup(func() { current.run(t, "flag", id, "--mailbox", current.source, "--clear") })

		reported := current.changes(t, "flag", id, "--mailbox", current.source, "--color", flagColor)
		if reported.ID != row.ID {
			t.Fatalf("flag reported id %d, want %d", reported.ID, row.ID)
		}
		if !current.read(t, id, current.source).Flagged {
			t.Fatalf("message %d is not flagged after flag --color %s", row.ID, flagColor)
		}
		current.changes(t, "flag", id, "--mailbox", current.source, "--clear")
		if current.read(t, id, current.source).Flagged {
			t.Fatalf("message %d is still flagged after flag --clear", row.ID)
		}
	})

	t.Run("move", func(t *testing.T) {
		if current.target == "" {
			t.Skipf("set %s to a mailbox to move a message to and back from", targetVariable)
		}
		listed := identified(t, current.messages(t))
		row := listed[len(listed)-1]
		moved := false
		t.Cleanup(func() {
			if moved {
				current.run(t, "move", row.MessageID, "--mailbox", current.target, "--to", current.source)
			}
		})

		there := current.changes(t, "move", row.MessageID, "--mailbox", current.source, "--to", current.target)
		moved = true
		if there.ID != row.ID || there.MessageID != row.MessageID {
			t.Fatalf("move reported id %d, want %d with the same Message-ID", there.ID, row.ID)
		}
		switch {
		case there.NewID == nil:
			t.Fatal("move reported no newId")
		case *there.NewID == 0:
			t.Log("move reported newId 0: Mail had not assigned the moved message an id in time")
		}
		current.run(t, "read", strconv.Itoa(row.ID), "--mailbox", current.source).expect(t, failure.NotFound)
		if found := current.read(t, row.MessageID, current.target); found.MessageID != row.MessageID {
			t.Fatalf("the moved message is not in %q", current.target)
		}

		back := current.changes(t, "move", row.MessageID, "--mailbox", current.target, "--to", current.source)
		moved = false
		if back.NewID == nil {
			t.Fatal("move back reported no newId")
		}
		current.read(t, row.MessageID, current.source)
	})

	t.Run("delete", func(t *testing.T) {
		listed := identified(t, current.messages(t))
		row := listed[0]
		trash := current.trash(t)
		deleted := false
		t.Cleanup(func() {
			if deleted {
				current.run(t, "move", row.MessageID, "--mailbox", trash, "--to", current.source)
			}
		})

		reported := current.changes(t, "delete", row.MessageID, "--mailbox", current.source)
		deleted = true
		if reported.MessageID != row.MessageID || reported.NewID == nil {
			t.Fatalf("delete reported Message-ID match %t, newId set %t", reported.MessageID == row.MessageID, reported.NewID != nil)
		}
		if found := current.read(t, row.MessageID, trash); !same(found.Mailbox, trash) {
			t.Fatalf("the deleted message is in %q, want %q", found.Mailbox, trash)
		}

		current.changes(t, "move", row.MessageID, "--mailbox", trash, "--to", current.source)
		deleted = false
		current.read(t, row.MessageID, current.source)
	})

	t.Run("archive dry run", func(t *testing.T) {
		before := current.messages(t)
		row := identified(t, before)[0]
		result := current.run(t, "archive", row.MessageID, "--mailbox", current.source, "--dry-run")
		if result.code == failure.NotFound && !continuous() {
			t.Skip(noArchive)
		}
		var rows []change
		result.expect(t, failure.Success).decode(t, &rows)
		if len(rows) != 1 || !rows[0].DryRun || rows[0].NewID != nil || rows[0].ID != row.ID {
			t.Fatalf("archive --dry-run reported %d rows, want one dry-run row for message %d without newId", len(rows), row.ID)
		}
		current.read(t, strconv.Itoa(row.ID), current.source)
		if after := current.messages(t); len(after) != len(before) {
			t.Fatalf("archive --dry-run changed the mailbox from %d to %d messages", len(before), len(after))
		}
	})
}

// TestAccount sends, waits for, replies to and merges mail through a real account. It only ever
// writes to the account's own address and puts the messages it made in Trash afterwards.
func TestAccount(t *testing.T) {
	current := begin(t)
	if current.account == mail.Local {
		t.Skip("local mailboxes cannot send; the account tests need a real account")
	}
	var accounts []account
	current.run(t, "accounts").expect(t, failure.Success).decode(t, &accounts)
	var own string
	for _, item := range accounts {
		if strings.EqualFold(item.Name, current.account) && len(item.Addresses) > 0 {
			own = item.Addresses[0]
		}
	}
	if own == "" {
		t.Fatalf("account %s has no address to send to itself", current.account)
	}
	token := fmt.Sprintf("%s%d", subjectPrefix, time.Now().UnixNano())
	t.Cleanup(func() { current.discard(t, token) })

	var sent summary
	t.Run("send and wait", func(t *testing.T) {
		waited := make(chan outcome, 1)
		go func() {
			waited <- current.run(t, "wait", "--subject", token, "--code", "--timeout", waitTimeout, "--interval", waitInterval)
		}()
		// wait only counts messages that were not there on its first look, so it has to look before the message is sent.
		time.Sleep(waitHeadStart)

		var delivered delivery
		current.run(t, "send", "--to", own, "--subject", token, "--body", codeBody, "--send").expect(t, failure.Success).decode(t, &delivered)
		if !delivered.Sent {
			t.Fatal("send --send reported the message as not sent")
		}

		var found arrival
		(<-waited).expect(t, failure.Success).decode(t, &found)
		if found.Code != expectedCode {
			t.Fatalf("wait --code found code %q, want %s", found.Code, expectedCode)
		}
		if found.Message.MessageID == "" || !strings.Contains(found.Message.Subject, token) {
			t.Fatalf("wait returned message %d without a Message-ID or the subject token", found.Message.ID)
		}
		sent = found.Message
		t.Logf("message %d arrived", sent.ID)
	})

	t.Run("reply", func(t *testing.T) {
		if sent.MessageID == "" {
			t.Skip("needs the message from send and wait")
		}
		// reply answers the Reply-To address, else the sender, so it is sent only when that is the account's own address.
		original := current.read(t, sent.MessageID, sent.Mailbox)
		recipient := valueOr(original.ReplyTo, original.Sender)
		if parsed, err := netmail.ParseAddress(recipient); err != nil || !strings.EqualFold(parsed.Address, own) {
			t.Fatal("the reply would not go to the account's own address; nothing was sent")
		}
		var replied delivery
		current.run(t, "reply", sent.MessageID, "--mailbox", sent.Mailbox, "--body", replyBody, "--send").expect(t, failure.Success).decode(t, &replied)
		if !replied.Sent {
			t.Fatal("reply --send reported the reply as not sent")
		}
	})

	t.Run("merge dry run", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), csvName)
		if err := os.WriteFile(path, []byte("email,name\n"+own+",E2E\n"), 0o600); err != nil {
			t.Fatalf("write CSV: %v", err)
		}
		var rows []merged
		current.run(t, "merge", path, "--to", "{{email}}", "--subject", token+" {{name}}", "--body", "Hello {{name}}", "--dry-run").expect(t, failure.Success).decode(t, &rows)
		if len(rows) != 1 || rows[0].Row != 1 || len(rows[0].To) != 1 || !strings.EqualFold(rows[0].To[0], own) || rows[0].Subject != token+" E2E" {
			t.Fatalf("merge --dry-run reported %d rows, want one row to the account's own address with the filled subject", len(rows))
		}
	})
}

// discard puts every message whose subject carries the token, received or sent, in Trash. Search lists
// one row per mailbox copy (Gmail shows a message in Inbox, All Mail and Sent Mail), so each Message-ID
// is deleted once, never through All Mail, and a copy that is already gone counts as handled.
func (current *session) discard(t *testing.T, token string) {
	t.Helper()
	trash := current.trash(t)
	var found []summary
	result := current.run(t, "search", token, "--all-mailboxes", "--limit", "0")
	if result.code != failure.Success {
		t.Errorf("cleanup could not search for the test messages; stderr: %s", strings.TrimSpace(result.stderr))
		return
	}
	result.decode(t, &found)
	handled := make(map[string]bool, len(found))
	removed := 0
	for _, row := range found {
		key := strings.ToLower(row.MessageID)
		if key == "" {
			key = strings.ToLower(row.Mailbox) + "/" + strconv.Itoa(row.ID)
		}
		if handled[key] || same(row.Mailbox, trash) || same(row.Mailbox, allMail) || !strings.Contains(row.Subject, token) {
			continue
		}
		switch result := current.run(t, "delete", strconv.Itoa(row.ID), "--mailbox", row.Mailbox); result.code {
		case failure.Success:
			removed++
		case failure.NotFound:
			t.Logf("cleanup found message %d of %q already gone", row.ID, row.Mailbox)
		default:
			t.Errorf("cleanup could not put message %d of %q in Trash; stderr: %s", row.ID, row.Mailbox, strings.TrimSpace(result.stderr))
			continue
		}
		handled[key] = true
	}
	t.Logf("cleanup put %d test messages in Trash", removed)
}
