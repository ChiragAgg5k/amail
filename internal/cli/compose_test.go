package cli_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/ChiragAgg5k/mailslot/internal/failure"
	"github.com/ChiragAgg5k/mailslot/internal/mail"
)

const (
	composeSubject   = "Launch checklist"
	composeRecipient = "robin@example.com"
	composeBody      = "Here is the checklist."
	composeAlias     = "sam@example.com"
	composeDefault   = "sam.rivera@example.com"
	composeDrafts    = "Drafts"
	composeSent      = "Sent"
)

// composeArguments are the arguments of a plain send to the Work account, followed by extra.
func composeArguments(extra ...string) []string {
	return append([]string{"send", "-a", "Work", "--to", composeRecipient, "--subject", composeSubject, "--body", composeBody}, extra...)
}

// composeFind reads the newest message with the subject from a Work mailbox, failing when there is none.
func composeFind(t *testing.T, mailbox, subject string) mail.Message {
	t.Helper()
	var summaries []mail.Summary
	execute(t, "list", "-a", "Work", "-m", mailbox, "-n", "0", "--json").succeeded(t).decode(t, &summaries)
	for _, summary := range summaries {
		if summary.Subject != subject {
			continue
		}
		var message mail.Message
		execute(t, "read", strconv.Itoa(summary.ID), "-a", "Work", "-m", mailbox, "--json").succeeded(t).decode(t, &message)
		return message
	}
	t.Fatalf("no message %q in Work %s", subject, mailbox)
	return mail.Message{}
}

// composeAbsent fails when a message with the subject sits in the Work mailbox.
func composeAbsent(t *testing.T, mailbox, subject string) {
	t.Helper()
	var summaries []mail.Summary
	execute(t, "list", "-a", "Work", "-m", mailbox, "-n", "0", "--json").succeeded(t).decode(t, &summaries)
	for _, summary := range summaries {
		if summary.Subject == subject {
			t.Fatalf("message %q unexpectedly in Work %s", subject, mailbox)
		}
	}
}

func composeExpect(t *testing.T, outcome result, code failure.Code) {
	t.Helper()
	if outcome.Code != int(code) {
		t.Fatalf("exit code %d, want %d\nstdout:\n%s\nstderr:\n%s", outcome.Code, code, outcome.Stdout, outcome.Stderr)
	}
}

func TestSendOpensDraft(t *testing.T) {
	outcome := execute(t, composeArguments()...).succeeded(t)
	if !strings.Contains(outcome.Stdout, "draft from "+composeDefault) || !strings.Contains(outcome.Stdout, "nothing was sent") {
		t.Fatalf("draft output:\n%s", outcome.Stdout)
	}
	draft := composeFind(t, composeDrafts, composeSubject)
	if draft.Sender != composeDefault || draft.Content != composeBody {
		t.Fatalf("draft %+v", draft)
	}
	if len(draft.To) != 1 || draft.To[0].Address != composeRecipient {
		t.Fatalf("draft recipients %+v", draft.To)
	}
	composeAbsent(t, composeSent, composeSubject)
}

func TestSendDelivers(t *testing.T) {
	outcome := execute(t, composeArguments("--send")...).succeeded(t)
	if !strings.Contains(outcome.Stdout, "Sent from "+composeDefault) {
		t.Fatalf("send output:\n%s", outcome.Stdout)
	}
	composeFind(t, composeSent, composeSubject)
	composeAbsent(t, composeDrafts, composeSubject)
}

func TestSendFromAlias(t *testing.T) {
	var delivery mail.Delivery
	execute(t, composeArguments("--from", composeAlias, "--json")...).succeeded(t).decode(t, &delivery)
	if delivery.Sent || delivery.Sender != composeAlias {
		t.Fatalf("delivery %+v", delivery)
	}
	if sender := composeFind(t, composeDrafts, composeSubject).Sender; sender != composeAlias {
		t.Fatalf("draft sender %q, want %q", sender, composeAlias)
	}
}

func TestSendFromDisplayName(t *testing.T) {
	var delivery mail.Delivery
	execute(t, composeArguments("--from", "Sam Rivera <"+composeAlias+">", "--send", "--json")...).succeeded(t).decode(t, &delivery)
	if !delivery.Sent || delivery.Sender != composeAlias {
		t.Fatalf("delivery %+v, want sent from %s", delivery, composeAlias)
	}
	if sender := composeFind(t, composeSent, composeSubject).Sender; sender != composeAlias {
		t.Fatalf("sent sender %q, want %q", sender, composeAlias)
	}
}

func TestSendFromForeignAddress(t *testing.T) {
	outcome := execute(t, composeArguments("--from", "sam.rivera@example.org", "--send")...)
	composeExpect(t, outcome, failure.Usage)
	for _, address := range []string{composeDefault, composeAlias} {
		if !strings.Contains(outcome.Stderr, address) {
			t.Errorf("hint lacks %s:\n%s", address, outcome.Stderr)
		}
	}
	composeAbsent(t, composeSent, composeSubject)
	composeAbsent(t, composeDrafts, composeSubject)
}

func TestSendSignature(t *testing.T) {
	execute(t, composeArguments("--signature", "Work")...).succeeded(t)
	content := composeFind(t, composeDrafts, composeSubject).Content
	if !strings.HasPrefix(content, composeBody) || !strings.Contains(content, "Engineering Lead, Example Co.") {
		t.Fatalf("draft content lacks the signature:\n%s", content)
	}
}

func TestSendUnknownSignature(t *testing.T) {
	composeExpect(t, execute(t, composeArguments("--signature", "Missing")...), failure.NotFound)
	composeAbsent(t, composeDrafts, composeSubject)
}

func TestSendNoSendEnvironment(t *testing.T) {
	switched := setup{Environment: map[string]string{"MAILSLOT_NO_SEND": "1"}}
	composeExpect(t, executeWith(t, switched, composeArguments("--send")...), failure.Refused)
	composeAbsent(t, composeSent, composeSubject)
	executeWith(t, switched, composeArguments()...).succeeded(t)
	composeFind(t, composeDrafts, composeSubject)
}

func TestSendNoSendConfig(t *testing.T) {
	prepare(t, setup{})
	path := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "mailslot", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("send = false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	composeExpect(t, execute(t, composeArguments("--send")...), failure.Refused)
	composeAbsent(t, composeSent, composeSubject)
	execute(t, composeArguments()...).succeeded(t)
	composeFind(t, composeDrafts, composeSubject)
}

func TestSendReadOnly(t *testing.T) {
	switched := setup{Environment: map[string]string{"MAILSLOT_READ_ONLY": "1"}}
	composeExpect(t, executeWith(t, switched, composeArguments()...), failure.Refused)
	composeExpect(t, executeWith(t, switched, "reply", "1001", "-a", "Work", "--body", "Yes"), failure.Refused)
	composeAbsent(t, composeDrafts, composeSubject)
}

func TestSendLocalAccount(t *testing.T) {
	composeExpect(t, execute(t, "send", "-a", mail.Local, "--to", composeRecipient, "--subject", composeSubject, "--body", composeBody), failure.Usage)
	composeExpect(t, execute(t, "reply", "1022", "-a", mail.Local, "-m", "Receipts", "--body", "Thanks"), failure.Usage)
}

func TestSendMissingBody(t *testing.T) {
	composeExpect(t, execute(t, "send", "-a", "Work", "--to", composeRecipient, "--subject", composeSubject), failure.Usage)
}

func TestReplyDraft(t *testing.T) {
	outcome := execute(t, "reply", "1001", "-a", "Work", "--body", "Thursday works.").succeeded(t)
	if !strings.Contains(outcome.Stdout, "nothing was sent") {
		t.Fatalf("reply output:\n%s", outcome.Stdout)
	}
	draft := composeFind(t, composeDrafts, "Re: Quarterly planning")
	if len(draft.To) != 1 || draft.To[0].Address != "dana.okafor@example.com" {
		t.Fatalf("reply recipients %+v", draft.To)
	}
	if len(draft.Cc) != 0 {
		t.Fatalf("reply without --all copied %+v", draft.Cc)
	}
	if !strings.HasPrefix(draft.Content, "Thursday works.") || !strings.Contains(draft.Content, "> Can we hold the quarterly planning session") {
		t.Fatalf("reply content lacks the quote:\n%s", draft.Content)
	}
}

func TestReplyAll(t *testing.T) {
	execute(t, "reply", "1001", "-a", "Work", "--all", "--body", "Thursday works.", "--from", composeAlias).succeeded(t)
	draft := composeFind(t, composeDrafts, "Re: Quarterly planning")
	if draft.Sender != composeAlias {
		t.Fatalf("reply sender %q", draft.Sender)
	}
	var copied []string
	for _, address := range draft.Cc {
		copied = append(copied, address.Address)
	}
	if got, want := strings.Join(copied, ","), "priya.lindqvist@example.com"; got != want {
		t.Fatalf("reply --all copied %s, want %s", got, want)
	}
}

func TestReplyFromForeignAddress(t *testing.T) {
	composeExpect(t, execute(t, "reply", "1001", "-a", "Work", "--body", "Yes", "--from", "someone@example.net"), failure.Usage)
}

func TestComposeJSON(t *testing.T) {
	var delivery map[string]any
	execute(t, composeArguments("--send", "--json")...).succeeded(t).decode(t, &delivery)
	if len(delivery) != 2 || delivery["sent"] != true || delivery["sender"] != composeDefault {
		t.Fatalf("delivery JSON %v", delivery)
	}
	var reply mail.Delivery
	execute(t, "reply", "1001", "-a", "Work", "--body", "Yes", "--json").succeeded(t).decode(t, &reply)
	if reply != (mail.Delivery{Sent: false, Sender: composeDefault}) {
		t.Fatalf("reply delivery %+v", reply)
	}
}

func TestComposeAttachment(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(path, []byte("notes"), 0o644); err != nil {
		t.Fatal(err)
	}
	execute(t, composeArguments("--attach", path)...).succeeded(t)
	attachments := composeFind(t, composeDrafts, composeSubject).Attachments
	if len(attachments) != 1 || attachments[0].Name != "notes.txt" {
		t.Fatalf("attachments %+v", attachments)
	}
	composeExpect(t, execute(t, composeArguments("--attach", filepath.Join(t.TempDir(), "missing.txt"))...), failure.Usage)
}
