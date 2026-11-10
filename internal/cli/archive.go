package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ChiragAgg5k/mailslot/internal/failure"
	"github.com/ChiragAgg5k/mailslot/internal/mail"
)

func (app *application) archive() *cobra.Command {
	var (
		mailbox string
		dryRun  bool
	)
	command := &cobra.Command{
		Use:   "archive <reference>...",
		Short: "Move messages to the account's archive mailbox",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(command *cobra.Command, references []string) error {
			return app.relocate(command, references, dryRun, app.special(mail.Archive), func(mailbox string) error {
				return failure.New(failure.Usage, fmt.Sprintf("the message is already in %s", mailbox), "")
			})
		},
	}
	mailboxFlag(command, &mailbox)
	dryRunFlag(command, &dryRun)
	return changing(command)
}
