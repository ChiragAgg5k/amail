package cli_test

import (
	"strings"
	"testing"

	"github.com/ChiragAgg5k/mailslot/internal/mail"
)

const waitCode = "482913"

type waitArrival struct {
	Message mail.Summary `json:"message"`
	Code    string       `json:"code"`
}

func TestWaitPrintsCode(t *testing.T) {
	outcome := execute(t, "wait", "-a", "Work", "--subject", "verification", "--code", "--interval", "10ms", "--timeout", "2s")
	outcome.succeeded(t)
	if got := strings.TrimSpace(outcome.Stdout); got != waitCode {
		t.Fatalf("stdout = %q, want %q", got, waitCode)
	}
}

func TestWaitJSON(t *testing.T) {
	outcome := execute(t, "wait", "-a", "Work", "--subject", "verification", "--code", "--interval", "10ms", "--timeout", "2s", "--json")
	outcome.succeeded(t)
	var arrival waitArrival
	outcome.decode(t, &arrival)
	if arrival.Code != waitCode {
		t.Fatalf("code = %q, want %q", arrival.Code, waitCode)
	}
	if arrival.Message.Subject != "Your verification code" || arrival.Message.Mailbox != mail.Inbox || arrival.Message.Read {
		t.Fatalf("message = %+v, want the unread verification message in Inbox", arrival.Message)
	}
}

func TestWaitTable(t *testing.T) {
	outcome := execute(t, "wait", "-a", "Work", "--from", "no-reply@example.com", "--interval", "10ms", "--timeout", "2s")
	outcome.succeeded(t)
	if !strings.Contains(outcome.Stdout, "Your verification code") || !strings.Contains(outcome.Stdout, "Example Sign-in") {
		t.Fatalf("stdout = %q, want the arrived message's row", outcome.Stdout)
	}
}

func TestWaitIgnoresExistingMessages(t *testing.T) {
	outcome := execute(t, "wait", "-a", "Work", "--subject", "Quarterly planning", "--interval", "10ms", "--timeout", "50ms")
	if outcome.Code != 5 {
		t.Fatalf("exit code = %d, want 5; stderr %q", outcome.Code, outcome.Stderr)
	}
}

func TestWaitTimeout(t *testing.T) {
	outcome := execute(t, "wait", "-a", "Work", "--subject", "nothing-matches", "--interval", "10ms", "--timeout", "50ms")
	if outcome.Code != 5 {
		t.Fatalf("exit code = %d, want 5; stderr %q", outcome.Code, outcome.Stderr)
	}
	if !strings.Contains(outcome.Stderr, "nothing matching arrived within 50ms") {
		t.Fatalf("stderr = %q, want the timeout message", outcome.Stderr)
	}
}

func TestWaitWithoutCode(t *testing.T) {
	outcome := execute(t, "wait", "-a", "Personal", "--from", "tomasz", "--code", "--interval", "10ms", "--timeout", "2s")
	if outcome.Code != 4 {
		t.Fatalf("exit code = %d, want 4; stderr %q", outcome.Code, outcome.Stderr)
	}
}

func TestWaitNeedsFromOrSubject(t *testing.T) {
	outcome := execute(t, "wait", "-a", "Work", "--timeout", "50ms")
	if outcome.Code != 2 {
		t.Fatalf("exit code = %d, want 2; stderr %q", outcome.Code, outcome.Stderr)
	}
}

func TestWaitRefusesLocal(t *testing.T) {
	outcome := execute(t, "wait", "-a", mail.Local, "--subject", "verification", "--timeout", "50ms")
	if outcome.Code != 2 {
		t.Fatalf("exit code = %d, want 2; stderr %q", outcome.Code, outcome.Stderr)
	}
}
