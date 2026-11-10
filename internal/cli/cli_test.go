package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ChiragAgg5k/mailslot/internal/cli"
	"github.com/ChiragAgg5k/mailslot/internal/mail"
)

const (
	testVersion     = "test"
	fixtureData     = "fixture"
	localData       = "local"
	fixtureVariable = "MAILSLOT_FIXTURE"
	stateVariable   = "MAILSLOT_FIXTURE_STATE"
	lockVariable    = "MAILSLOT_LOCK_TIMEOUT"
	lockTimeout     = "200ms"
)

// cleared are the variables a developer's shell may set that would change what a command does.
var cleared = []string{
	"MAILSLOT_ACCOUNT",
	"MAILSLOT_READ_ONLY",
	"MAILSLOT_NO_SEND",
	"MAILSLOT_FIXTURE_FAULT",
}

// result is what one mailslot invocation produced.
type result struct {
	Code   int
	Stdout string
	Stderr string
}

// setup varies how a command runs. The zero value runs against testdata/fixture with this test's state file.
type setup struct {
	// Data names the fixture directory under testdata: fixtureData (default) or localData.
	Data string
	// Environment sets extra variables after the defaults, so it can also override them.
	Environment map[string]string
	// Input is fed to the command's standard input.
	Input string
}

// sandbox holds the directories one test's commands share, so later commands see what earlier ones changed.
type sandbox struct {
	home   string
	config string
	cache  string
	path   string
	state  string
}

var sandboxes sync.Map

func TestMain(m *testing.M) {
	time.Local = time.UTC
	os.Exit(m.Run())
}

func sandboxOf(t *testing.T) *sandbox {
	t.Helper()
	if existing, found := sandboxes.Load(t); found {
		return existing.(*sandbox)
	}
	root := t.TempDir()
	created := &sandbox{
		home:   filepath.Join(root, "home"),
		config: filepath.Join(root, "config"),
		cache:  filepath.Join(root, "cache"),
		path:   filepath.Join(root, "bin"),
		state:  filepath.Join(root, "state.json"),
	}
	for _, directory := range []string{created.home, created.config, created.cache, created.path} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatalf("create sandbox: %v", err)
		}
	}
	sandboxes.Store(t, created)
	t.Cleanup(func() { sandboxes.Delete(t) })
	return created
}

// prepare sets the environment a command sees. Tests that hold the Mail lock call it before mail.LockPath so both agree on the cache directory.
func prepare(t *testing.T, options setup) {
	t.Helper()
	data := options.Data
	if data == "" {
		data = fixtureData
	}
	directory, err := filepath.Abs(filepath.Join("..", "..", "testdata", data))
	if err != nil {
		t.Fatalf("locate fixture data: %v", err)
	}
	box := sandboxOf(t)
	for _, name := range cleared {
		t.Setenv(name, "")
		os.Unsetenv(name)
	}
	t.Setenv(fixtureVariable, directory)
	t.Setenv(stateVariable, box.state)
	t.Setenv("HOME", box.home)
	t.Setenv("XDG_CONFIG_HOME", box.config)
	t.Setenv("XDG_CACHE_HOME", box.cache)
	t.Setenv("PATH", box.path)
	t.Setenv(lockVariable, lockTimeout)
	for name, value := range options.Environment {
		t.Setenv(name, value)
	}
}

// execute runs mailslot against testdata/fixture.
func execute(t *testing.T, arguments ...string) result {
	t.Helper()
	return executeWith(t, setup{}, arguments...)
}

// executeWith runs mailslot with a varied setup.
func executeWith(t *testing.T, options setup, arguments ...string) result {
	t.Helper()
	prepare(t, options)
	var stdout, stderr bytes.Buffer
	code := cli.Run(testVersion, arguments, strings.NewReader(options.Input), &stdout, &stderr)
	return result{Code: code, Stdout: stdout.String(), Stderr: stderr.String()}
}

func (outcome result) succeeded(t *testing.T) result {
	t.Helper()
	if outcome.Code != 0 {
		t.Fatalf("exit code %d\nstdout:\n%s\nstderr:\n%s", outcome.Code, outcome.Stdout, outcome.Stderr)
	}
	return outcome
}

func (outcome result) decode(t *testing.T, target any) {
	t.Helper()
	if err := json.Unmarshal([]byte(outcome.Stdout), target); err != nil {
		t.Fatalf("decode output: %v\n%s", err, outcome.Stdout)
	}
}

func TestSmokeAccounts(t *testing.T) {
	var accounts []mail.Account
	execute(t, "accounts", "--json").succeeded(t).decode(t, &accounts)
	var names []string
	for _, account := range accounts {
		names = append(names, account.Name)
	}
	if got, want := strings.Join(names, ","), "Work,Personal,Old,local"; got != want {
		t.Fatalf("accounts %s, want %s", got, want)
	}
	if last := accounts[len(accounts)-1]; last.Kind != mail.LocalKind {
		t.Fatalf("local account kind %q", last.Kind)
	}
}

func TestSmokeAccountsWithoutRealAccounts(t *testing.T) {
	var accounts []mail.Account
	executeWith(t, setup{Data: localData}, "accounts", "--json").succeeded(t).decode(t, &accounts)
	if len(accounts) != 1 || accounts[0].Name != mail.Local {
		t.Fatalf("accounts %+v, want only local", accounts)
	}
}

func TestSmokeMailboxes(t *testing.T) {
	outcome := execute(t, "mailboxes", "-a", "Work").succeeded(t)
	for _, name := range []string{"Inbox", "Archive", "Drafts", "Sent", "Trash", "Projects"} {
		if !strings.Contains(outcome.Stdout, name) {
			t.Errorf("mailboxes output lacks %s:\n%s", name, outcome.Stdout)
		}
	}
}

func TestSmokeList(t *testing.T) {
	outcome := execute(t, "list", "-a", "Work").succeeded(t)
	if !strings.Contains(outcome.Stdout, "Quarterly planning") {
		t.Fatalf("list output lacks the planning thread:\n%s", outcome.Stdout)
	}
	var summaries []mail.Summary
	execute(t, "list", "-a", "Work", "--json").succeeded(t).decode(t, &summaries)
	if len(summaries) == 0 {
		t.Fatal("list returned no messages")
	}
	for index, summary := range summaries {
		if summary.Mailbox != mail.Inbox || summary.MessageID == "" {
			t.Errorf("summary %+v lacks its mailbox or Message-ID", summary)
		}
		if index > 0 && summary.Received.After(*summaries[index-1].Received) {
			t.Errorf("list not newest first at %d", index)
		}
	}
}
