package cli

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/ChiragAgg5k/mailslot/internal/mail"
)

const defaultLimit = 20

func (app *application) list() *cobra.Command {
	var mailbox string
	var limit int
	var unread bool
	command := &cobra.Command{
		Use:   "list",
		Short: "Show the newest messages of a mailbox",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			account, err := app.resolve(command.Context())
			if err != nil {
				return err
			}
			location := mail.Location{Account: account.Name, Mailbox: mailbox}
			rows, err := app.backend.List(command.Context(), location, limit, unread)
			if err != nil {
				return err
			}
			return app.print(command.OutOrStdout(), rows, func(output io.Writer) { summaries(output, rows) })
		},
	}
	mailboxFlag(command, &mailbox)
	command.Flags().IntVarP(&limit, "limit", "n", defaultLimit, "how many messages to show (0 for all)")
	command.Flags().BoolVar(&unread, "unread", false, "only unread messages")
	return command
}
