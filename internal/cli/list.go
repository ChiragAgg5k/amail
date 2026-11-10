package cli

import (
	"io"

	"github.com/spf13/cobra"
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
			_, location, err := app.locate(command.Context(), mailbox)
			if err != nil {
				return err
			}
			rows, err := app.client.List(command.Context(), location, limit, unread)
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
