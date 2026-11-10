package cli

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ChiragAgg5k/mailslot/internal/config"
	"github.com/ChiragAgg5k/mailslot/internal/failure"
	"github.com/ChiragAgg5k/mailslot/internal/lock"
	"github.com/ChiragAgg5k/mailslot/internal/mail"
)

const (
	// preparesItself marks a command that root's prepare leaves alone, because it reports what prepare would abort on.
	preparesItself = "mailslot.preparesItself"
	osascriptName  = "osascript"
)

const (
	checkOsascript  = "osascript"
	checkMail       = "mail"
	checkAutomation = "automation"
	checkAccounts   = "accounts"
	checkConfig     = "config"
	checkSwitches   = "switches"
)

const (
	switchReadOnly    = "read_only"
	switchNoSend      = "no-send"
	sourceConfig      = "config"
	sourceEnvironment = "environment"
)

func (app *application) doctor() *cobra.Command {
	return &cobra.Command{
		Use:         "doctor",
		Short:       "Check that Mail is present, answering and permitted, without changing anything",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{preparesItself: annotationMarker},
		RunE: func(command *cobra.Command, _ []string) error {
			found := app.diagnose(command.Context())
			if err := app.print(command.OutOrStdout(), found, found.table); err != nil {
				return err
			}
			return found.verdict()
		},
	}
}

// diagnose runs every check. It never stops early, so one problem does not hide the others.
func (app *application) diagnose(ctx context.Context) diagnosis {
	app.fixture = os.Getenv(fixtureVariable)
	configuration, stored := configCheck()
	app.config = stored
	app.switches = stored.Switches()

	tool := app.osascriptCheck()
	checks := []check{tool}
	if tool.Status == checkFailed {
		reason := "needs " + osascriptName
		checks = append(checks, skipped(checkMail, reason), skipped(checkAutomation, reason), skipped(checkAccounts, reason))
	} else {
		checks = append(checks, app.mailChecks(ctx)...)
	}
	return diagnosis{Checks: append(checks, configuration, switchesCheck(stored))}
}

func (app *application) osascriptCheck() check {
	if app.fixture != "" {
		return skipped(checkOsascript, "fixture backend at "+app.fixture)
	}
	if runtime.GOOS != supportedSystem {
		return failed(checkOsascript, failure.New(failure.General, "mailslot drives the Apple Mail app, so it only runs on macOS", ""))
	}
	path, err := exec.LookPath(osascriptName)
	if err != nil {
		return failed(checkOsascript, failure.Wrap(failure.General, fmt.Errorf("find %s: %w", osascriptName, err), "osascript ships with macOS in /usr/bin; make sure /usr/bin is on PATH"))
	}
	return passed(checkOsascript, path)
}

// mailChecks asks Mail for a real Apple event (accounts.length, never only the version, which answers without permission).
func (app *application) mailChecks(ctx context.Context) []check {
	backend, err := app.open()
	if err != nil {
		reason := "Mail could not be reached"
		return []check{failed(checkMail, err), skipped(checkAutomation, reason), skipped(checkAccounts, reason)}
	}
	app.backend = backend

	status, err := backend.Ping(ctx)
	if err != nil {
		reason := "needs Mail to answer"
		if errors.Is(err, lock.ErrHeld) {
			unknown := "unknown while another mailslot command holds the lock"
			return []check{failed(checkMail, err), skipped(checkAutomation, unknown), skipped(checkAccounts, reason)}
		}
		switch failure.CodeOf(err) {
		case failure.Permission:
			return []check{skipped(checkMail, "no answer; see "+checkAutomation), failed(checkAutomation, err), skipped(checkAccounts, reason)}
		case failure.Busy:
			busy := failure.New(failure.Busy, "Mail is busy or did not answer", failure.HintOf(err))
			return []check{failed(checkMail, busy), skipped(checkAutomation, "unknown while Mail is not answering"), skipped(checkAccounts, reason)}
		}
		return []check{failed(checkMail, err), skipped(checkAutomation, reason), skipped(checkAccounts, reason)}
	}
	return []check{
		passed(checkMail, fmt.Sprintf("Mail %s answered", status.Version)),
		passed(checkAutomation, "this process may control Mail"),
		app.accountsCheck(ctx),
	}
}

func (app *application) accountsCheck(ctx context.Context) check {
	accounts, err := app.backend.Accounts(ctx)
	if err != nil {
		return failed(checkAccounts, err)
	}
	mailboxes, err := app.backend.Mailboxes(ctx, mail.Local)
	if err != nil {
		return failed(checkAccounts, err)
	}
	local := fmt.Sprintf("%d local mailboxes", len(mailboxes))
	if len(accounts) == 0 {
		return warned(checkAccounts, "no accounts in Mail; "+local, "use --account "+mail.Local+" for the On My Mac mailboxes, or add an account in System Settings → Internet Accounts")
	}
	names := make([]string, 0, len(accounts))
	for _, account := range accounts {
		name := account.Name
		if !account.Enabled {
			name += " (disabled)"
		}
		names = append(names, name)
	}
	return passed(checkAccounts, strings.Join(names, ", ")+"; "+local)
}

// configCheck reports where the configuration lives and whether it loads; a broken file falls back to defaults.
func configCheck() (check, config.Config) {
	path, err := config.Path()
	if err != nil {
		return failed(checkConfig, failure.Wrap(failure.General, fmt.Errorf("locate configuration: %w", err), "set HOME or XDG_CONFIG_HOME")), config.Config{}
	}
	stored, err := config.Load()
	if err != nil {
		return failed(checkConfig, failure.Wrap(failure.General, fmt.Errorf("read %s: %w", path, err), "fix or remove "+path)), config.Config{}
	}
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return passed(checkConfig, path+" (not present, using defaults)"), stored
	}
	return passed(checkConfig, path+" (loaded)"), stored
}

// switchesCheck reports each safety switch and what turned it on; an empty configuration isolates the environment's part.
func switchesCheck(stored config.Config) check {
	combined := stored.Switches()
	environment := config.Config{}.Switches()
	return passed(checkSwitches, strings.Join([]string{
		describeSwitch(switchReadOnly, combined.ReadOnly, stored.ReadOnly, environment.ReadOnly),
		describeSwitch(switchNoSend, combined.NoSend, stored.Send != nil && !*stored.Send, environment.NoSend),
	}, "; "))
}

func describeSwitch(name string, on, fromConfig, fromEnvironment bool) string {
	if !on {
		return name + " off"
	}
	var sources []string
	if fromConfig {
		sources = append(sources, sourceConfig)
	}
	if fromEnvironment {
		sources = append(sources, sourceEnvironment)
	}
	return fmt.Sprintf("%s on (%s)", name, strings.Join(sources, ", "))
}
