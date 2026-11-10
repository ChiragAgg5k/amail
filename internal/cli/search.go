package cli

import (
	"cmp"
	"fmt"
	"io"
	"slices"
	"time"

	"github.com/spf13/cobra"

	"github.com/ChiragAgg5k/mailslot/internal/failure"
	"github.com/ChiragAgg5k/mailslot/internal/mail"
)

const (
	dateLayout        = "2006-01-02"
	allMailboxesName  = "all-mailboxes"
	bodyName          = "body"
	sinceName         = "since"
	searchDescription = `Find messages by subject, sender, date or unread state.

The text matches the subject or the sender. Message bodies are searched only with --body, which
reads every candidate message one by one, so it needs --since to narrow the search.

One mailbox is searched by default (Inbox). Repeat --mailbox to search several, or use
--all-mailboxes for every mailbox of the account; they are searched one after another and the
results are merged, newest first.`
)

func (app *application) search() *cobra.Command {
	var since string
	var mailboxes []string
	var everywhere bool
	query := mail.Query{}
	command := &cobra.Command{
		Use:   "search [text]",
		Short: "Find messages by subject, sender, date or unread state",
		Long:  searchDescription,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(command *cobra.Command, arguments []string) error {
			if len(arguments) == 1 {
				query.Text = arguments[0]
			}
			if since != "" {
				moment, err := time.ParseInLocation(dateLayout, since, time.Local)
				if err != nil {
					return failure.New(failure.Usage, fmt.Sprintf("--%s wants a date such as %s", sinceName, dateLayout), "")
				}
				query.Since = &moment
			}
			if query.Text == "" && query.From == "" && query.Subject == "" && !query.Unread && query.Since == nil {
				return failure.New(failure.Usage, "give some text or at least one of --from, --subject, --since, --unread", "")
			}
			if query.Body && (query.Text == "" || query.Since == nil) {
				return failure.New(failure.Usage, "--body needs search text and --since", "bodies are read one message at a time, so narrow the search with --since")
			}
			if everywhere && command.Flags().Changed(mailboxName) {
				return failure.New(failure.Usage, fmt.Sprintf("--%s and --%s cannot be used together", allMailboxesName, mailboxName), usageHint(command))
			}
			account, err := app.resolve(command.Context())
			if err != nil {
				return err
			}
			if everywhere {
				found, err := app.backend.Mailboxes(command.Context(), account.Name)
				if err != nil {
					return err
				}
				mailboxes = make([]string, 0, len(found))
				for _, mailbox := range found {
					mailboxes = append(mailboxes, mailbox.Name)
				}
			}
			rows, err := app.searchMailboxes(command, query, account.Name, mailboxes)
			if err != nil {
				return err
			}
			return app.print(command.OutOrStdout(), rows, func(output io.Writer) { summaries(output, rows) })
		},
	}
	command.Flags().StringArrayVarP(&mailboxes, mailboxName, "m", []string{mail.Inbox}, "mailbox to look in (repeat for several)")
	command.Flags().BoolVar(&everywhere, allMailboxesName, false, "look in every mailbox of the account")
	command.Flags().StringVar(&query.From, "from", "", "sender contains this text")
	command.Flags().StringVar(&query.Subject, "subject", "", "subject contains this text")
	command.Flags().StringVar(&since, sinceName, "", "received on or after this date (YYYY-MM-DD)")
	command.Flags().BoolVar(&query.Unread, "unread", false, "only unread messages")
	command.Flags().BoolVar(&query.Body, bodyName, false, "also match the text in message bodies (slow; needs --since)")
	command.Flags().IntVarP(&query.Limit, "limit", "n", defaultLimit, "how many messages to show (0 for all)")
	return command
}

// searchMailboxes runs the query in each mailbox in turn, because Mail answers one request at a time,
// then merges the matches newest first and applies the limit across all of them.
func (app *application) searchMailboxes(command *cobra.Command, query mail.Query, account string, mailboxes []string) ([]mail.Summary, error) {
	rows := []mail.Summary{}
	for _, mailbox := range slices.Compact(mailboxes) {
		query.Location = mail.Location{Account: account, Mailbox: mailbox}
		found, err := app.backend.Search(command.Context(), query)
		if err != nil {
			return nil, fmt.Errorf("search %s: %w", mailbox, err)
		}
		rows = append(rows, found...)
	}
	slices.SortStableFunc(rows, newestFirst)
	if query.Limit > 0 && len(rows) > query.Limit {
		rows = rows[:query.Limit]
	}
	return rows, nil
}

func newestFirst(first, second mail.Summary) int {
	switch {
	case first.Received == nil && second.Received == nil:
		return 0
	case first.Received == nil:
		return 1
	case second.Received == nil:
		return -1
	}
	return cmp.Compare(second.Received.UnixNano(), first.Received.UnixNano())
}
