package cli

import (
	"github.com/spf13/cobra"

	"github.com/ChiragAgg5k/mailslot/internal/failure"
	"github.com/ChiragAgg5k/mailslot/internal/mail"
)

// delete only ever moves messages into Trash; mailslot has no permanent delete.
func (app *application) delete() *cobra.Command {
	var (
		mailbox string
		dryRun  bool
	)
	command := &cobra.Command{
		Use:   "delete <reference>...",
		Short: "Move messages to the account's Trash; nothing is erased",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(command *cobra.Command, references []string) error {
			return app.relocate(command, references, dryRun, app.special(mail.Trash), func(string) error {
				return failure.New(failure.Usage, "already in Trash; mailslot never deletes permanently", "empty the Trash in Mail if you mean to erase it")
			})
		},
	}
	mailboxFlag(command, &mailbox)
	dryRunFlag(command, &dryRun)
	return changing(command)
}
