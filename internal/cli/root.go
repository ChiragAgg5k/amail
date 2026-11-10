// Package cli defines the amail command line.
package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ChiragAgg5k/amail/internal/config"
	"github.com/ChiragAgg5k/amail/internal/mail"
)

const (
	accountVariable = "AMAIL_ACCOUNT"
	supportedSystem = "darwin"
)

type application struct {
	client  mail.Client
	account string
	json    bool
}

// Execute runs the command line and returns the process exit code.
func Execute(version string) int {
	app := &application{}
	root := &cobra.Command{
		Use:           "amail",
		Short:         "Apple Mail from the terminal",
		Long:          "amail reads, searches and writes mail through the Apple Mail app, so it works with any account Mail can sign in to.",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVarP(&app.account, "account", "a", "", "Mail account to use (default: "+accountVariable+", then the configured account)")
	root.PersistentFlags().BoolVar(&app.json, "json", false, "print JSON instead of a table")
	root.AddCommand(
		app.accounts(),
		app.mailboxes(),
		app.list(),
		app.read(),
		app.search(),
		app.send(),
		app.reply(),
		app.attachments(),
	)
	root.PersistentPreRunE = func(*cobra.Command, []string) error {
		if runtime.GOOS != supportedSystem {
			return errors.New("amail drives the Apple Mail app, so it only runs on macOS")
		}
		return nil
	}
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "amail:", err)
		return 1
	}
	return 0
}

// resolve picks the account to act on: the flag, the environment, the configuration
// file, and finally the only enabled account when there is exactly one.
func (app *application) resolve(ctx context.Context) (mail.Account, error) {
	accounts, err := app.client.Accounts(ctx)
	if err != nil {
		return mail.Account{}, err
	}
	name := app.account
	if name == "" {
		name = os.Getenv(accountVariable)
	}
	if name == "" {
		stored, err := config.Load()
		if err != nil {
			return mail.Account{}, fmt.Errorf("read configuration: %w", err)
		}
		name = stored.Account
	}

	var enabled, names []string
	for _, account := range accounts {
		names = append(names, account.Name)
		if account.Enabled {
			enabled = append(enabled, account.Name)
		}
		if name != "" && strings.EqualFold(account.Name, name) {
			return account, nil
		}
	}
	switch {
	case name != "":
		return mail.Account{}, fmt.Errorf("no account named %q in Mail; available: %s", name, strings.Join(names, ", "))
	case len(accounts) == 0:
		return mail.Account{}, errors.New("Mail has no accounts; add one in System Settings → Internet Accounts")
	case len(enabled) == 1:
		for _, account := range accounts {
			if account.Name == enabled[0] {
				return account, nil
			}
		}
	}
	return mail.Account{}, fmt.Errorf("several accounts are available (%s); pick one with --account or `amail accounts use <name>`", strings.Join(names, ", "))
}

func (app *application) locate(ctx context.Context, mailbox string) (mail.Account, mail.Location, error) {
	account, err := app.resolve(ctx)
	if err != nil {
		return account, mail.Location{}, err
	}
	return account, mail.Location{Account: account.Name, Mailbox: mailbox}, nil
}

func mailboxFlag(command *cobra.Command, target *string) {
	command.Flags().StringVarP(target, "mailbox", "m", mail.Inbox, "mailbox to look in")
}
