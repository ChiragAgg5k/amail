package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func (app *application) mailboxes() *cobra.Command {
	return &cobra.Command{
		Use:   "mailboxes",
		Short: "List the mailboxes of the account",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			account, err := app.resolve(command.Context())
			if err != nil {
				return err
			}
			mailboxes, err := app.backend.Mailboxes(command.Context(), account.Name)
			if err != nil {
				return err
			}
			return app.print(command.OutOrStdout(), mailboxes, func(output io.Writer) {
				columns(output, "MAILBOX\tUNREAD\tMESSAGES", func(writer io.Writer) {
					for _, mailbox := range mailboxes {
						fmt.Fprintf(writer, "%s\t%d\t%d\n", mailbox.Name, mailbox.Unread, mailbox.Messages)
					}
				})
			})
		},
	}
}
