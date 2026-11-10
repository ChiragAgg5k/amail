// Package cli defines the mailslot command line.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/ChiragAgg5k/mailslot/internal/config"
	"github.com/ChiragAgg5k/mailslot/internal/failure"
	"github.com/ChiragAgg5k/mailslot/internal/mail"
)

const (
	accountVariable  = "MAILSLOT_ACCOUNT"
	supportedSystem  = "darwin"
	jsonName         = "json"
	mailboxName      = "mailbox"
	dryRunName       = "dry-run"
	sendName         = "send"
	argumentsEnd     = "--"
	changesMail      = "mailslot.changesMail"
	annotationMarker = "true"
	// boolType is what pflag reports as the type of a boolean flag.
	boolType = "bool"
)

type application struct {
	backend  mail.Backend
	account  string
	json     bool
	switches config.Switches
	fixture  string
	config   config.Config
	// running is set once a command's RunE starts; untyped errors before that are usage errors.
	running bool
}

// Execute runs the command line against the process's own streams and returns the exit code.
func Execute(version string) int {
	return Run(version, os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
}

// Run runs the command line with the given arguments and streams and returns the exit code.
func Run(version string, arguments []string, input io.Reader, output, errors io.Writer) int {
	app := &application{}
	root := app.root(version)
	root.SetArgs(arguments)
	root.SetIn(input)
	root.SetOut(output)
	root.SetErr(errors)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	command, err := root.ExecuteContextC(ctx)
	if err == nil {
		return int(failure.Success)
	}
	err = app.classify(command, err)
	report(errors, err, app.json || requested(arguments))
	return int(failure.CodeOf(err))
}

func (app *application) root(version string) *cobra.Command {
	root := &cobra.Command{
		Use:               "mailslot",
		Short:             "Apple Mail from the terminal",
		Long:              "mailslot reads, searches and writes mail through the Apple Mail app, so it works with any account Mail can sign in to.",
		Version:           version,
		SilenceUsage:      true,
		SilenceErrors:     true,
		PersistentPreRunE: app.prepare,
	}
	root.PersistentFlags().StringVarP(&app.account, "account", "a", "", "Mail account to use (default: "+accountVariable+", then the configured account; "+mail.Local+" for On My Mac)")
	root.PersistentFlags().BoolVar(&app.json, jsonName, false, "print JSON instead of a table")
	root.SetFlagErrorFunc(func(command *cobra.Command, err error) error {
		return failure.Wrap(failure.Usage, err, usageHint(command))
	})
	root.AddCommand(
		app.accountsCommand(),
		app.mailboxes(),
		app.list(),
		app.read(),
		app.search(),
		app.send(),
		app.reply(),
		app.attachments(),
		app.mark(),
		app.flag(),
		app.move(),
		app.archive(),
		app.delete(),
		app.doctor(),
		app.wait(),
		app.merge(),
	)
	app.track(root)
	return root
}

// track wraps every RunE so that Run can tell errors raised while running from command line mistakes.
func (app *application) track(command *cobra.Command) {
	if run := command.RunE; run != nil {
		command.RunE = func(command *cobra.Command, arguments []string) error {
			app.running = true
			return run(command, arguments)
		}
	}
	for _, child := range command.Commands() {
		app.track(child)
	}
}

// prepare runs before every command: it checks the platform, applies the safety switches and opens the backend.
func (app *application) prepare(command *cobra.Command, _ []string) error {
	if command.Annotations[preparesItself] == annotationMarker {
		return nil
	}
	app.fixture = os.Getenv(fixtureVariable)
	if app.fixture == "" && runtime.GOOS != supportedSystem {
		return failure.New(failure.General, "mailslot drives the Apple Mail app, so it only runs on macOS", "")
	}
	stored, err := config.Load()
	if err != nil {
		path, _ := config.Path()
		return failure.Wrap(failure.General, fmt.Errorf("read configuration: %w", err), "fix or remove "+path)
	}
	app.config = stored
	app.switches = stored.Switches()
	if err := app.guard(command); err != nil {
		return err
	}
	backend, err := app.open()
	if err != nil {
		return err
	}
	app.backend = backend
	return nil
}

// guard refuses commands that the safety switches rule out.
func (app *application) guard(command *cobra.Command) error {
	if app.switches.ReadOnly && alters(command) && !enabled(command, dryRunName) {
		return failure.New(
			failure.Refused,
			fmt.Sprintf("mailslot is read-only (read_only in the config or MAILSLOT_READ_ONLY); %s would change Mail", command.Name()),
			"use --dry-run to preview, or remove the switch",
		)
	}
	if app.switches.NoSend && enabled(command, sendName) {
		return failure.New(
			failure.Refused,
			"sending is switched off (send = false in the config or MAILSLOT_NO_SEND)",
			"leave out --send to open a draft for review",
		)
	}
	return nil
}

// changing marks a command as one that changes Mail, so the read-only switch applies to it.
func changing(command *cobra.Command) *cobra.Command {
	if command.Annotations == nil {
		command.Annotations = map[string]string{}
	}
	command.Annotations[changesMail] = annotationMarker
	return command
}

func alters(command *cobra.Command) bool {
	return command.Annotations[changesMail] == annotationMarker
}

func enabled(command *cobra.Command, name string) bool {
	flag := command.Flags().Lookup(name)
	return flag != nil && flag.Value.Type() == boolType && flag.Value.String() == strconv.FormatBool(true)
}

// classify makes any untyped error raised before a command started running a usage error.
func (app *application) classify(command *cobra.Command, err error) error {
	var typed *failure.Error
	if app.running || errors.As(err, &typed) {
		return err
	}
	return failure.Wrap(failure.Usage, err, usageHint(command))
}

func usageHint(command *cobra.Command) string {
	if command == nil {
		return "run `mailslot --help` for usage"
	}
	return fmt.Sprintf("run `%s --help` for usage", command.CommandPath())
}

// requested finds --json on the raw command line, so errors raised while parsing it still come out as JSON.
func requested(arguments []string) bool {
	flag := "--" + jsonName
	for _, argument := range arguments {
		if argument == argumentsEnd {
			return false
		}
		if argument == flag {
			return true
		}
		if value, found := strings.CutPrefix(argument, flag+"="); found {
			parsed, err := strconv.ParseBool(value)
			return err == nil && parsed
		}
	}
	return false
}

// accounts lists the real accounts followed by the local pseudo-account.
func (app *application) accounts(ctx context.Context) ([]mail.Account, error) {
	accounts, err := app.backend.Accounts(ctx)
	if err != nil {
		return nil, err
	}
	return append(accounts, mail.LocalAccount()), nil
}

// resolve picks the account to act on: the flag, the environment, the configuration file, then the
// only enabled account, or the local mailboxes when Mail has no accounts. The name local is reserved.
func (app *application) resolve(ctx context.Context) (mail.Account, error) {
	name := app.account
	if name == "" {
		name = os.Getenv(accountVariable)
	}
	if name == "" {
		name = app.config.Account
	}
	if name == mail.Local {
		return mail.LocalAccount(), nil
	}
	accounts, err := app.backend.Accounts(ctx)
	if err != nil {
		return mail.Account{}, err
	}

	var enabled []mail.Account
	names := make([]string, 0, len(accounts)+1)
	for _, account := range accounts {
		names = append(names, account.Name)
		if account.Enabled {
			enabled = append(enabled, account)
		}
		if name != "" && strings.EqualFold(account.Name, name) {
			return account, nil
		}
	}
	names = append(names, mail.Local)
	switch {
	case name != "":
		return mail.Account{}, failure.New(failure.NotFound, fmt.Sprintf("no account named %q in Mail", name), "available: "+strings.Join(names, ", "))
	case len(enabled) == 1:
		return enabled[0], nil
	case len(accounts) == 0:
		return mail.LocalAccount(), nil
	}
	return mail.Account{}, failure.New(
		failure.Usage,
		fmt.Sprintf("several accounts are available (%s)", strings.Join(names, ", ")),
		"pick one with --account or `mailslot accounts use <name>`",
	)
}

func mailboxFlag(command *cobra.Command, target *string) {
	command.Flags().StringVarP(target, mailboxName, "m", mail.Inbox, "mailbox to look in")
}
