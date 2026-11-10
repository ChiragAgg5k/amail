package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/ChiragAgg5k/mailslot/internal/failure"
	"github.com/ChiragAgg5k/mailslot/internal/mail"
)

const dateLayout = "2006-01-02"

func (app *application) search() *cobra.Command {
	var mailbox, since string
	query := mail.Query{}
	command := &cobra.Command{
		Use:   "search [text]",
		Short: "Find messages by subject, sender, date or unread state",
		Long:  "Find messages in one mailbox. The text matches the subject or the sender; message bodies are not searched.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(command *cobra.Command, arguments []string) error {
			if len(arguments) == 1 {
				query.Text = arguments[0]
			}
			if since != "" {
				moment, err := time.ParseInLocation(dateLayout, since, time.Local)
				if err != nil {
					return failure.New(failure.Usage, fmt.Sprintf("--since wants a date such as %s", dateLayout), "")
				}
				query.Since = &moment
			}
			if query.Text == "" && query.From == "" && query.Subject == "" && !query.Unread && query.Since == nil {
				return failure.New(failure.Usage, "give some text or at least one of --from, --subject, --since, --unread", "")
			}
			account, err := app.resolve(command.Context())
			if err != nil {
				return err
			}
			query.Location = mail.Location{Account: account.Name, Mailbox: mailbox}
			rows, err := app.backend.Search(command.Context(), query)
			if err != nil {
				return err
			}
			return app.print(command.OutOrStdout(), rows, func(output io.Writer) { summaries(output, rows) })
		},
	}
	mailboxFlag(command, &mailbox)
	command.Flags().StringVar(&query.From, "from", "", "sender contains this text")
	command.Flags().StringVar(&query.Subject, "subject", "", "subject contains this text")
	command.Flags().StringVar(&since, "since", "", "received on or after this date (YYYY-MM-DD)")
	command.Flags().BoolVar(&query.Unread, "unread", false, "only unread messages")
	command.Flags().IntVarP(&query.Limit, "limit", "n", defaultLimit, "how many messages to show (0 for all)")
	return command
}
