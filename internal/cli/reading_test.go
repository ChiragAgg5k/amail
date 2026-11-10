package cli_test

import (
	"context"
	"encoding/json"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ChiragAgg5k/mailslot/internal/lock"
	"github.com/ChiragAgg5k/mailslot/internal/mail"
)

const (
	readingFaultVariable = "MAILSLOT_FIXTURE_FAULT"
	readingPlanning      = "quarterly-1@mail.example.com"
	readingNonASCII      = "Café rendezvous in Zürich — 東京 ✈"
)

// readingProblem is the JSON error mailslot writes to standard error.
type readingProblem struct {
	Error string `json:"error"`
	Code  int    `json:"code"`
	Hint  string `json:"hint"`
}

// readingFailed checks the exit code and decodes the JSON error from standard error.
func readingFailed(t *testing.T, outcome result, code int) readingProblem {
	t.Helper()
	if outcome.Code != code {
		t.Fatalf("exit code %d, want %d\nstdout:\n%s\nstderr:\n%s", outcome.Code, code, outcome.Stdout, outcome.Stderr)
	}
	var problem readingProblem
	if err := json.Unmarshal([]byte(outcome.Stderr), &problem); err != nil {
		t.Fatalf("decode error: %v\n%s", err, outcome.Stderr)
	}
	if problem.Error == "" || problem.Code != code {
		t.Fatalf("error %+v, want code %d and a message", problem, code)
	}
	return problem
}

func readingIDs(summaries []mail.Summary) []int {
	ids := make([]int, 0, len(summaries))
	for _, summary := range summaries {
		ids = append(ids, summary.ID)
	}
	return ids
}

func readingHas(ids []int, id int) bool {
	for _, candidate := range ids {
		if candidate == id {
			return true
		}
	}
	return false
}

func TestReadingAccountsIncludeLocal(t *testing.T) {
	outcome := execute(t, "accounts").succeeded(t)
	for _, name := range []string{"Work", "Personal", "Old", mail.Local, "sam.rivera@example.com"} {
		if !strings.Contains(outcome.Stdout, name) {
			t.Errorf("accounts output lacks %s:\n%s", name, outcome.Stdout)
		}
	}
}

func TestReadingDefaultAccount(t *testing.T) {
	readingFailed(t, execute(t, "list", "--json"), 2)
	outcome := execute(t, "accounts", "use", "work").succeeded(t)
	if !strings.Contains(outcome.Stdout, "Work") {
		t.Fatalf("accounts use output %q", outcome.Stdout)
	}
	var summaries []mail.Summary
	execute(t, "list", "--json").succeeded(t).decode(t, &summaries)
	if !readingHas(readingIDs(summaries), 1001) {
		t.Fatalf("list after accounts use lacks 1001: %v", readingIDs(summaries))
	}
}

func TestReadingLocalWithoutAccounts(t *testing.T) {
	var summaries []mail.Summary
	executeWith(t, setup{Data: localData}, "list", "-m", "Receipts", "--json").succeeded(t).decode(t, &summaries)
	if ids := readingIDs(summaries); len(ids) != 1 || ids[0] != 2001 {
		t.Fatalf("local Receipts %v, want [2001]", ids)
	}
	outcome := executeWith(t, setup{Data: localData}, "mailboxes").succeeded(t)
	for _, name := range []string{"Receipts", "Archive", "Import/fixture"} {
		if !strings.Contains(outcome.Stdout, name) {
			t.Errorf("local mailboxes lack %s:\n%s", name, outcome.Stdout)
		}
	}
}

func TestReadingUnknownAccount(t *testing.T) {
	problem := readingFailed(t, execute(t, "list", "-a", "Nope", "--json"), 4)
	if !strings.Contains(problem.Hint, "Work") || !strings.Contains(problem.Hint, mail.Local) {
		t.Fatalf("hint %q does not list the accounts", problem.Hint)
	}
}

func TestReadingMailboxes(t *testing.T) {
	var mailboxes []mail.Mailbox
	execute(t, "mailboxes", "-a", "Work", "--json").succeeded(t).decode(t, &mailboxes)
	if len(mailboxes) != 6 || mailboxes[0].Name != mail.Inbox {
		t.Fatalf("Work mailboxes %+v", mailboxes)
	}
	outcome := execute(t, "mailboxes", "-a", mail.Local).succeeded(t)
	for _, name := range []string{"Receipts", "Archive", "Trash", "Import/fixture"} {
		if !strings.Contains(outcome.Stdout, name) {
			t.Errorf("local mailboxes lack %s:\n%s", name, outcome.Stdout)
		}
	}
}

func TestReadingListUnreadAndLimit(t *testing.T) {
	var unread []mail.Summary
	execute(t, "list", "-a", "Work", "--unread", "--json").succeeded(t).decode(t, &unread)
	if len(unread) == 0 {
		t.Fatal("no unread messages")
	}
	for _, summary := range unread {
		if summary.Read {
			t.Errorf("--unread listed read message %d", summary.ID)
		}
	}
	if !readingHas(readingIDs(unread), 1003) {
		t.Errorf("--unread lacks 1003: %v", readingIDs(unread))
	}
	var limited []mail.Summary
	execute(t, "list", "-a", "Work", "-n", "2", "--json").succeeded(t).decode(t, &limited)
	if ids := readingIDs(limited); len(ids) != 2 || ids[0] != 1010 || ids[1] != 1009 {
		t.Fatalf("-n 2 gave %v, want [1010 1009]", ids)
	}
}

func TestReadByReference(t *testing.T) {
	for _, reference := range []string{"1001", "<" + readingPlanning + ">", readingPlanning} {
		var message mail.Message
		execute(t, "read", reference, "-a", "Work", "--json").succeeded(t).decode(t, &message)
		if message.ID != 1001 || message.MessageID != readingPlanning || message.Mailbox != mail.Inbox {
			t.Errorf("read %s gave %d %s in %s", reference, message.ID, message.MessageID, message.Mailbox)
		}
	}
	outcome := execute(t, "read", "1001", "-a", "Work").succeeded(t)
	for _, part := range []string{"From:    Dana Okafor", "Subject: Quarterly planning", "Cc:"} {
		if !strings.Contains(outcome.Stdout, part) {
			t.Errorf("read output lacks %q:\n%s", part, outcome.Stdout)
		}
	}
}

func TestReadNumericMiss(t *testing.T) {
	problem := readingFailed(t, execute(t, "read", "9999", "-a", "Work", "--json"), 4)
	if !strings.Contains(problem.Hint, "Message-ID") {
		t.Fatalf("hint %q does not suggest retrying with the Message-ID", problem.Hint)
	}
}

func TestReadRaw(t *testing.T) {
	plain := execute(t, "read", "1001", "-a", "Work", "--raw").succeeded(t).Stdout
	for _, header := range []string{"From: \"Dana Okafor\" <dana.okafor@example.com>\r\n", "Subject: Quarterly planning\r\n", "Message-ID: <" + readingPlanning + ">\r\n"} {
		if !strings.Contains(plain, header) {
			t.Errorf("raw source lacks %q:\n%s", header, plain)
		}
	}

	encoded := execute(t, "read", "1007", "-a", "Personal", "--raw").succeeded(t).Stdout
	subject := ""
	for _, line := range strings.Split(encoded, "\r\n") {
		if value, found := strings.CutPrefix(line, "Subject: "); found {
			subject = value
		}
	}
	if !strings.HasPrefix(subject, "=?utf-8?") {
		t.Fatalf("non-ASCII subject is not RFC 2047 encoded: %q", subject)
	}
	decoded, err := new(mime.WordDecoder).DecodeHeader(subject)
	if err != nil || decoded != readingNonASCII {
		t.Fatalf("subject decodes to %q (%v), want %q", decoded, err, readingNonASCII)
	}

	multipart := execute(t, "read", "1004", "-a", "Work", "--raw").succeeded(t).Stdout
	for _, part := range []string{"Content-Type: multipart/mixed; boundary=", "filename=agenda.txt", "filename=budget.csv"} {
		if !strings.Contains(multipart, part) {
			t.Errorf("multipart source lacks %q", part)
		}
	}

	var described struct {
		ID        int    `json:"id"`
		MessageID string `json:"messageId"`
		Mailbox   string `json:"mailbox"`
		Source    string `json:"source"`
	}
	execute(t, "read", readingPlanning, "-a", "Work", "--raw", "--json").succeeded(t).decode(t, &described)
	if described.ID != 1001 || described.MessageID != readingPlanning || described.Mailbox != mail.Inbox || described.Source != plain {
		t.Fatalf("read --raw --json gave %+v", described)
	}
}

func TestReadRawOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "planning.eml")
	outcome := execute(t, "read", "1001", "-a", "Work", "--raw", "-o", path).succeeded(t)
	if strings.TrimSpace(outcome.Stdout) != path {
		t.Fatalf("read -o printed %q, want %q", outcome.Stdout, path)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved source: %v", err)
	}
	if source := execute(t, "read", "1001", "-a", "Work", "--raw").succeeded(t).Stdout; string(saved) != source {
		t.Fatalf("saved source differs from printed source")
	}
	readingFailed(t, execute(t, "read", "1001", "-a", "Work", "-o", path, "--json"), 2)
}

func TestReadAttachmentsOutput(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "saved")
	var saved []mail.SavedAttachment
	execute(t, "attachments", "1004", "-a", "Work", "-o", directory, "--json").succeeded(t).decode(t, &saved)
	if len(saved) != 2 {
		t.Fatalf("saved %+v, want two attachments", saved)
	}
	for _, attachment := range saved {
		if filepath.Dir(attachment.Path) != directory {
			t.Errorf("attachment saved to %s, want inside %s", attachment.Path, directory)
		}
		if _, err := os.Stat(attachment.Path); err != nil {
			t.Errorf("attachment %s missing: %v", attachment.Name, err)
		}
	}
}

func TestSearchText(t *testing.T) {
	var summaries []mail.Summary
	execute(t, "search", "planning", "-a", "Work", "--json").succeeded(t).decode(t, &summaries)
	if ids := readingIDs(summaries); len(ids) != 2 || ids[0] != 1003 || ids[1] != 1001 {
		t.Fatalf("search planning gave %v, want [1003 1001]", ids)
	}
}

func TestSearchFromAndSince(t *testing.T) {
	var from []mail.Summary
	execute(t, "search", "--from", "dana", "-a", "Work", "--json").succeeded(t).decode(t, &from)
	for _, summary := range from {
		if !strings.Contains(strings.ToLower(summary.Sender), "dana") {
			t.Errorf("--from dana matched %s", summary.Sender)
		}
	}
	if len(from) != 3 {
		t.Errorf("--from dana gave %v, want three messages", readingIDs(from))
	}
	var since []mail.Summary
	execute(t, "search", "--since", "2026-10-10", "-a", "Work", "--json").succeeded(t).decode(t, &since)
	if ids := readingIDs(since); len(ids) != 3 || readingHas(ids, 1006) {
		t.Fatalf("--since 2026-10-10 gave %v, want [1010 1009 1008]", ids)
	}
	readingFailed(t, execute(t, "search", "--since", "10/10/2026", "-a", "Work", "--json"), 2)
}

func TestSearchSeveralMailboxes(t *testing.T) {
	var summaries []mail.Summary
	execute(t, "search", "planning", "-a", "Work", "-m", "Inbox", "-m", "Sent", "--json").succeeded(t).decode(t, &summaries)
	if ids := readingIDs(summaries); len(ids) != 3 || ids[0] != 1003 || ids[1] != 1002 || ids[2] != 1001 {
		t.Fatalf("search across Inbox and Sent gave %v, want [1003 1002 1001]", ids)
	}
	outcome := execute(t, "search", "planning", "-a", "Work", "-m", "Inbox", "-m", "Sent").succeeded(t)
	if !strings.Contains(outcome.Stdout, "MAILBOX") || !strings.Contains(outcome.Stdout, "Sent") {
		t.Fatalf("table lacks the mailbox column:\n%s", outcome.Stdout)
	}
}

func TestSearchAllMailboxes(t *testing.T) {
	var summaries []mail.Summary
	execute(t, "search", "--since", "2025-01-01", "-a", "Work", "--all-mailboxes", "-n", "0", "--json").succeeded(t).decode(t, &summaries)
	ids := readingIDs(summaries)
	for _, id := range []int{1001, 1002, 1012, 1013, 1014, 1015, 1016} {
		if !readingHas(ids, id) {
			t.Errorf("--all-mailboxes lacks %d: %v", id, ids)
		}
	}
	for index := 1; index < len(summaries); index++ {
		if summaries[index].Received.After(*summaries[index-1].Received) {
			t.Fatalf("results not newest first at %d: %v", index, ids)
		}
	}
	var limited []mail.Summary
	execute(t, "search", "--since", "2025-01-01", "-a", "Work", "--all-mailboxes", "-n", "3", "--json").succeeded(t).decode(t, &limited)
	if got := readingIDs(limited); len(got) != 3 || got[0] != ids[0] || got[2] != ids[2] {
		t.Fatalf("limit across mailboxes gave %v, want the newest three of %v", got, ids)
	}
	readingFailed(t, execute(t, "search", "planning", "-a", "Work", "--all-mailboxes", "-m", "Sent", "--json"), 2)
}

func TestSearchBody(t *testing.T) {
	var without []mail.Summary
	execute(t, "search", "INV-2041", "-a", "Work", "--json").succeeded(t).decode(t, &without)
	if len(without) != 0 {
		t.Fatalf("search without --body matched bodies: %v", readingIDs(without))
	}
	var with []mail.Summary
	execute(t, "search", "INV-2041", "--body", "--since", "2026-01-01", "-a", "Work", "--all-mailboxes", "--json").succeeded(t).decode(t, &with)
	if ids := readingIDs(with); len(ids) != 1 || ids[0] != 1008 {
		t.Fatalf("--body gave %v, want [1008]", ids)
	}
	readingFailed(t, execute(t, "search", "INV-2041", "--body", "-a", "Work", "--json"), 2)
	readingFailed(t, execute(t, "search", "--body", "--since", "2026-01-01", "-a", "Work", "--json"), 2)
}

func TestSearchNonASCII(t *testing.T) {
	var summaries []mail.Summary
	execute(t, "search", "zürich", "-a", "Personal", "--json").succeeded(t).decode(t, &summaries)
	if len(summaries) != 1 || summaries[0].ID != 1007 || summaries[0].Subject != readingNonASCII {
		t.Fatalf("search zürich gave %+v", summaries)
	}
}

func TestErrorsUsage(t *testing.T) {
	readingFailed(t, execute(t, "bogus", "--json"), 2)
	problem := readingFailed(t, execute(t, "read", "not-a-reference", "-a", "Work", "--json"), 2)
	if problem.Hint == "" {
		t.Fatal("bad reference has no hint")
	}
}

func TestErrorsFaults(t *testing.T) {
	cases := map[string]int{"permission": 3, "timeout": 5}
	for fault, code := range cases {
		t.Run(fault, func(t *testing.T) {
			outcome := executeWith(t, setup{Environment: map[string]string{readingFaultVariable: fault}}, "list", "-a", "Work", "--json")
			if problem := readingFailed(t, outcome, code); problem.Hint == "" {
				t.Fatalf("%s error has no hint: %+v", fault, problem)
			}
		})
	}
}

func TestErrorsLockHeld(t *testing.T) {
	prepare(t, setup{})
	path, err := mail.LockPath()
	if err != nil {
		t.Fatalf("lock path: %v", err)
	}
	held, err := lock.Acquire(context.Background(), path, time.Second)
	if err != nil {
		t.Fatalf("hold lock: %v", err)
	}
	defer held.Release()
	problem := readingFailed(t, executeWith(t, setup{Environment: map[string]string{lockVariable: "150ms"}}, "list", "-a", "Work", "--json"), 5)
	if problem.Hint == "" {
		t.Fatalf("busy error has no hint: %+v", problem)
	}
}
