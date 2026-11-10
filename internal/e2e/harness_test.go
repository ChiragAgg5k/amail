//go:build e2e

package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ChiragAgg5k/mailslot/internal/cli"
	"github.com/ChiragAgg5k/mailslot/internal/failure"
	"github.com/ChiragAgg5k/mailslot/internal/mail"
)

const (
	accountVariable  = "MAILSLOT_E2E_ACCOUNT"
	mailboxVariable  = "MAILSLOT_E2E_MAILBOX"
	targetVariable   = "MAILSLOT_E2E_TARGET"
	markerVariable   = "MAILSLOT_E2E_MARKER"
	fixtureVariable  = "MAILSLOT_FIXTURE"
	readOnlyVariable = "MAILSLOT_READ_ONLY"
	noSendVariable   = "MAILSLOT_NO_SEND"
	configVariable   = "XDG_CONFIG_HOME"
	ciVariable       = "CI"
	defaultMarker    = "mailslot-e2e-marker"
	version          = "e2e"
	jsonFlag         = "--json"
	pollInterval     = 5 * time.Second
	arrivalTimeout   = 60 * time.Second
	listLimit        = "20"
)

// TestMain refuses the fixture backend, which would make the suite test nothing, and runs every
// command with an empty configuration so a stored default account or read_only setting cannot change
// what is tested.
func TestMain(m *testing.M) {
	if os.Getenv(fixtureVariable) != "" {
		fmt.Fprintf(os.Stderr, "e2e: %s is set; the end-to-end suite only runs against the real Mail app\n", fixtureVariable)
		os.Exit(1)
	}
	if os.Getenv(accountVariable) != "" {
		for _, variable := range []string{readOnlyVariable, noSendVariable} {
			if os.Getenv(variable) != "" {
				fmt.Fprintf(os.Stderr, "e2e: %s is set; unset it for the end-to-end suite, which changes mail on purpose\n", variable)
				os.Exit(1)
			}
		}
	}
	directory, err := os.MkdirTemp("", "mailslot-e2e-config-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "e2e: create configuration directory: %v\n", err)
		os.Exit(1)
	}
	if err := os.Setenv(configVariable, directory); err != nil {
		fmt.Fprintf(os.Stderr, "e2e: set %s: %v\n", configVariable, err)
		os.Exit(1)
	}
	code := m.Run()
	os.RemoveAll(directory)
	os.Exit(code)
}

// session is the account and mailboxes the suite works in.
type session struct {
	account string
	// source is the mailbox the read and triage tests take messages from, as Mail names it.
	source string
	// target is the mailbox messages are moved to and back from, as Mail names it.
	target string
}

// begin skips unless the suite is pointed at an account and resolves the mailbox names it uses.
func begin(t *testing.T) *session {
	t.Helper()
	account := os.Getenv(accountVariable)
	if account == "" {
		t.Skipf("set %s to a dedicated account (or %s on the hosted CI runner) to run the end-to-end suite", accountVariable, mail.Local)
	}
	current := &session{account: account}
	var mailboxes []mailbox
	current.run(t, "mailboxes").expect(t, failure.Success).decode(t, &mailboxes)
	current.source = find(t, mailboxes, valueOr(os.Getenv(mailboxVariable), mail.Inbox))
	if wanted := os.Getenv(targetVariable); wanted != "" {
		current.target = find(t, mailboxes, wanted)
	}
	t.Logf("account %s: %d mailboxes; source %q, target %q", account, len(mailboxes), current.source, current.target)
	return current
}

// find picks the mailbox named exactly, or else one nested under a parent such as Import/fixture.
func find(t *testing.T, mailboxes []mailbox, wanted string) string {
	t.Helper()
	for _, candidate := range mailboxes {
		if strings.EqualFold(candidate.Name, wanted) {
			return candidate.Name
		}
	}
	for _, candidate := range mailboxes {
		if strings.HasSuffix(strings.ToLower(candidate.Name), "/"+strings.ToLower(wanted)) {
			return candidate.Name
		}
	}
	t.Fatalf("mailbox %q is not among the %d mailboxes of the account", wanted, len(mailboxes))
	return ""
}

// same reports whether two names are one mailbox, since Mail may name a nested mailbox by its leaf alone.
func same(first, second string) bool {
	first, second = strings.ToLower(first), strings.ToLower(second)
	return first == second || strings.HasSuffix(first, "/"+second) || strings.HasSuffix(second, "/"+first)
}

func valueOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func continuous() bool {
	return os.Getenv(ciVariable) != ""
}

// outcome is what one mailslot command printed and how it exited.
type outcome struct {
	arguments []string
	code      failure.Code
	stdout    string
	stderr    string
}

// run runs mailslot in-process against the session's account, asking for JSON.
func (current *session) run(t *testing.T, arguments ...string) outcome {
	t.Helper()
	return current.runWith(t, "", arguments...)
}

func (current *session) runWith(t *testing.T, input string, arguments ...string) outcome {
	t.Helper()
	full := append(append([]string{}, arguments...), "--account", current.account, jsonFlag)
	var stdout, stderr bytes.Buffer
	code := cli.Run(version, full, strings.NewReader(input), &stdout, &stderr)
	return outcome{arguments: full, code: failure.Code(code), stdout: stdout.String(), stderr: stderr.String()}
}

// expect fails the test unless the command exited with the code. Only the error report is logged,
// never standard output, which may hold message text.
func (result outcome) expect(t *testing.T, code failure.Code) outcome {
	t.Helper()
	if result.code != code {
		t.Fatalf("mailslot %s exited %d, want %d; stderr: %s", result.command(), result.code, code, strings.TrimSpace(result.stderr))
	}
	return result
}

func (result outcome) decode(t *testing.T, value any) {
	t.Helper()
	if err := json.Unmarshal([]byte(result.stdout), value); err != nil {
		t.Fatalf("mailslot %s printed output that is not the expected JSON (%d bytes): %v", result.command(), len(result.stdout), err)
	}
}

// command names the subcommand only, so logs never repeat subjects or addresses passed as flags.
func (result outcome) command() string {
	if len(result.arguments) == 0 {
		return ""
	}
	return result.arguments[0]
}

// messages lists the newest messages of the source mailbox.
func (current *session) messages(t *testing.T) []summary {
	t.Helper()
	var rows []summary
	current.run(t, "list", "--mailbox", current.source, "--limit", listLimit).expect(t, failure.Success).decode(t, &rows)
	return rows
}

// arrived polls the source mailbox until it holds messages, since imported mail shows up gradually.
func (current *session) arrived(t *testing.T) []summary {
	t.Helper()
	deadline := time.Now().Add(arrivalTimeout)
	for {
		rows := current.messages(t)
		if len(rows) > 0 {
			return rows
		}
		if time.Now().After(deadline) {
			t.Fatalf("mailbox %q held no messages after %s", current.source, arrivalTimeout)
		}
		time.Sleep(pollInterval)
	}
}

// identified picks the listed messages that carry a Message-ID, the reference that survives moves.
func identified(t *testing.T, rows []summary) []summary {
	t.Helper()
	found := make([]summary, 0, len(rows))
	for _, row := range rows {
		if row.MessageID != "" {
			found = append(found, row)
		}
	}
	if len(found) == 0 {
		t.Fatalf("none of the %d listed messages has a Message-ID", len(rows))
	}
	return found
}

// read reads one message of a mailbox by numeric id or Message-ID.
func (current *session) read(t *testing.T, reference, mailbox string) message {
	t.Helper()
	var found message
	current.run(t, "read", reference, "--mailbox", mailbox).expect(t, failure.Success).decode(t, &found)
	return found
}

// trash finds the account's Trash the way the CLI does.
func (current *session) trash(t *testing.T) string {
	t.Helper()
	var mailboxes []mailbox
	current.run(t, "mailboxes").expect(t, failure.Success).decode(t, &mailboxes)
	converted := make([]mail.Mailbox, 0, len(mailboxes))
	for _, item := range mailboxes {
		converted = append(converted, mail.Mailbox{Name: item.Name, Unread: item.Unread, Messages: item.Messages})
	}
	name, found := mail.Trash.Find(converted)
	if !found {
		t.Fatalf("found no Trash mailbox among the %d mailboxes of the account", len(mailboxes))
	}
	return name
}

// changes runs a triage command and returns its report, checking it covers exactly one message.
func (current *session) changes(t *testing.T, arguments ...string) change {
	t.Helper()
	var rows []change
	current.run(t, arguments...).expect(t, failure.Success).decode(t, &rows)
	if len(rows) != 1 {
		t.Fatalf("mailslot %s reported %d changes, want 1", arguments[0], len(rows))
	}
	return rows[0]
}
