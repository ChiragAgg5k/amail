package cli_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ChiragAgg5k/mailslot/internal/lock"
	"github.com/ChiragAgg5k/mailslot/internal/mail"
)

const doctorFault = "MAILSLOT_FIXTURE_FAULT"

type doctorCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
	Hint   string `json:"hint"`
}

type doctorReport struct {
	Checks []doctorCheck `json:"checks"`
}

// doctorChecks runs doctor --json and returns its checks by name, failing unless it exits with code.
func doctorChecks(t *testing.T, options setup, code int) map[string]doctorCheck {
	t.Helper()
	outcome := executeWith(t, options, "doctor", "--json")
	if outcome.Code != code {
		t.Fatalf("exit code %d, want %d\nstdout:\n%s\nstderr:\n%s", outcome.Code, code, outcome.Stdout, outcome.Stderr)
	}
	var report doctorReport
	outcome.decode(t, &report)
	checks := make(map[string]doctorCheck, len(report.Checks))
	var names []string
	for _, item := range report.Checks {
		checks[item.Name] = item
		names = append(names, item.Name)
	}
	if got, want := strings.Join(names, ","), "osascript,mail,automation,accounts,config,switches"; got != want {
		t.Fatalf("checks %s, want %s", got, want)
	}
	return checks
}

func TestDoctorTable(t *testing.T) {
	outcome := execute(t, "doctor").succeeded(t)
	for _, want := range []string{"CHECK", "STATUS", "DETAIL", "osascript", "skip", "fixture backend at", "Work, Personal, Old (disabled)", "read_only off"} {
		if !strings.Contains(outcome.Stdout, want) {
			t.Errorf("doctor output lacks %q:\n%s", want, outcome.Stdout)
		}
	}
	if strings.Contains(outcome.Stdout, "fail") {
		t.Errorf("doctor reports a failure:\n%s", outcome.Stdout)
	}
}

func TestDoctorJSON(t *testing.T) {
	checks := doctorChecks(t, setup{}, 0)
	for name, want := range map[string]string{"osascript": "skip", "mail": "ok", "automation": "ok", "accounts": "ok", "config": "ok", "switches": "ok"} {
		if got := checks[name].Status; got != want {
			t.Errorf("%s status %q, want %q", name, got, want)
		}
	}
	if detail := checks["accounts"].Detail; !strings.Contains(detail, "Work") || !strings.Contains(detail, "4 local mailboxes") {
		t.Errorf("accounts detail %q", detail)
	}
	if detail := checks["config"].Detail; !strings.Contains(detail, "not present") {
		t.Errorf("config detail %q", detail)
	}
}

func TestDoctorPermission(t *testing.T) {
	checks := doctorChecks(t, setup{Environment: map[string]string{doctorFault: "permission"}}, 3)
	automation := checks["automation"]
	if automation.Status != "fail" || !strings.Contains(automation.Hint, "Automation") {
		t.Fatalf("automation check %+v", automation)
	}
	if checks["accounts"].Status != "skip" {
		t.Errorf("accounts check %+v, want skip", checks["accounts"])
	}

	outcome := executeWith(t, setup{Environment: map[string]string{doctorFault: "permission"}}, "doctor")
	if outcome.Code != 3 || !strings.Contains(outcome.Stdout, "hint: allow it under System Settings") {
		t.Fatalf("exit code %d\nstdout:\n%s", outcome.Code, outcome.Stdout)
	}
}

func TestDoctorTimeout(t *testing.T) {
	checks := doctorChecks(t, setup{Environment: map[string]string{doctorFault: "timeout"}}, 5)
	if mailCheck := checks["mail"]; mailCheck.Status != "fail" || mailCheck.Detail != "Mail is busy or did not answer" || !strings.Contains(mailCheck.Hint, "permission prompt") {
		t.Fatalf("mail check %+v", mailCheck)
	}
	if automation := checks["automation"]; automation.Status != "skip" || !strings.Contains(automation.Detail, "not answering") {
		t.Fatalf("automation check %+v, want skipped while Mail is not answering", automation)
	}
}

func TestDoctorLockHeld(t *testing.T) {
	prepare(t, setup{})
	path, err := mail.FixtureLockPath()
	if err != nil {
		t.Fatalf("lock path: %v", err)
	}
	held, err := lock.Acquire(context.Background(), path, time.Second)
	if err != nil {
		t.Fatalf("hold lock: %v", err)
	}
	defer held.Release()
	checks := doctorChecks(t, setup{Environment: map[string]string{lockVariable: "150ms"}}, 5)
	if mailCheck := checks["mail"]; mailCheck.Status != "fail" || !strings.Contains(mailCheck.Detail, "another mailslot command") {
		t.Fatalf("mail check %+v", mailCheck)
	}
	if automation := checks["automation"]; automation.Status != "skip" || !strings.Contains(automation.Detail, "holds the lock") {
		t.Fatalf("automation check %+v, want skipped while the lock is held", automation)
	}
}

func TestDoctorWithoutAccounts(t *testing.T) {
	checks := doctorChecks(t, setup{Data: localData}, 0)
	accounts := checks["accounts"]
	if accounts.Status != "warn" || !strings.Contains(accounts.Hint, "--account local") {
		t.Fatalf("accounts check %+v", accounts)
	}
}

func TestDoctorReadOnly(t *testing.T) {
	checks := doctorChecks(t, setup{Environment: map[string]string{"MAILSLOT_READ_ONLY": "1"}}, 0)
	if detail := checks["switches"].Detail; !strings.Contains(detail, "read_only on (environment)") || !strings.Contains(detail, "no-send off") {
		t.Fatalf("switches detail %q", detail)
	}
}

func TestDoctorBrokenConfig(t *testing.T) {
	prepare(t, setup{})
	path := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "mailslot", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("read_only = maybe\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	checks := doctorChecks(t, setup{}, 1)
	if config := checks["config"]; config.Status != "fail" || !strings.Contains(config.Hint, "fix or remove") {
		t.Fatalf("config check %+v", config)
	}
	if checks["mail"].Status != "ok" {
		t.Errorf("mail check %+v, want ok despite the broken configuration", checks["mail"])
	}
}
