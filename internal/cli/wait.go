package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/ChiragAgg5k/mailslot/internal/failure"
	"github.com/ChiragAgg5k/mailslot/internal/mail"
)

const (
	defaultWaitTimeout  = 5 * time.Minute
	defaultWaitInterval = 10 * time.Second
	// lookback widens the search past the start so a Mail search by date never drops a message received today.
	lookback = 24 * time.Hour
	// grace accepts a message the server received shortly before wait started that Mail had not fetched yet.
	grace = 2 * time.Minute
)

// arrival is what wait prints: the message that arrived and, with --code, its one-time code.
type arrival struct {
	Message mail.Summary `json:"message"`
	Code    string       `json:"code,omitempty"`
}

func (app *application) wait() *cobra.Command {
	var mailbox string
	var timeout, interval time.Duration
	var code bool
	query := mail.Query{}
	command := &cobra.Command{
		Use:   "wait",
		Short: "Wait for a new message from a sender or with a subject to arrive",
		Long: "Check for new mail every --interval until a message matching --from or --subject arrives in the mailbox, or --timeout passes. " +
			"Only messages that were not there on the first check count, and none received more than 2 minutes before wait started. " +
			"With --code, print just the one-time code found in the message. " +
			"Wait still runs under the read-only switch: it asks Mail to check for new mail but changes no message.",
		Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			if query.From == "" && query.Subject == "" {
				return failure.New(failure.Usage, "give at least one of --from, --subject", usageHint(command))
			}
			if timeout <= 0 || interval <= 0 {
				return failure.New(failure.Usage, "--timeout and --interval must be positive durations", "use a Go duration such as 10s or 5m")
			}
			account, err := app.resolve(command.Context())
			if err != nil {
				return err
			}
			if account.Name == mail.Local {
				return failure.New(failure.Usage, "wait checks an account for new mail, and "+mail.Local+" is not an account", "pick an account with --account")
			}
			query.Location = mail.Location{Account: account.Name, Mailbox: mailbox}
			found, err := app.await(command.Context(), query, timeout, interval)
			if err != nil {
				return err
			}
			result := arrival{Message: found}
			if code {
				result.Code, err = app.extract(command.Context(), query.Location, found)
				if err != nil {
					return err
				}
				if !app.json {
					fmt.Fprintln(command.OutOrStdout(), result.Code)
					return nil
				}
			}
			return app.print(command.OutOrStdout(), result, func(output io.Writer) { summaries(output, []mail.Summary{found}) })
		},
	}
	mailboxFlag(command, &mailbox)
	command.Flags().StringVar(&query.From, "from", "", "sender contains this text")
	command.Flags().StringVar(&query.Subject, "subject", "", "subject contains this text")
	command.Flags().DurationVar(&timeout, "timeout", defaultWaitTimeout, "how long to wait before giving up")
	command.Flags().DurationVar(&interval, "interval", defaultWaitInterval, "how often to check for new mail")
	command.Flags().BoolVar(&code, "code", false, "print only the one-time code found in the message")
	return command
}

// await polls until a matching message arrives that was not there on the first poll. The receive date is the
// server's arrival time, which can predate the start when Mail fetches late, so only a message received before
// the grace window is ignored. Each backend call takes the Mail lock on its own, so other commands run between polls.
func (app *application) await(ctx context.Context, query mail.Query, timeout, interval time.Duration) (mail.Summary, error) {
	start := time.Now().Truncate(time.Second)
	earliest := start.Add(-grace)
	since := start.Add(-lookback)
	query.Since = &since
	bounded, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	expired := func(err error) error {
		if ctx.Err() == nil && errors.Is(bounded.Err(), context.DeadlineExceeded) {
			return failure.New(failure.Busy, fmt.Sprintf("nothing matching arrived within %s", timeout), "raise --timeout, or check --from, --subject and --mailbox")
		}
		return err
	}

	existing, err := app.backend.Search(bounded, query)
	if err != nil {
		return mail.Summary{}, expired(err)
	}
	seen := make(map[int]struct{}, len(existing))
	for _, row := range existing {
		seen[row.ID] = struct{}{}
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := app.backend.Fetch(bounded, query.Account); err != nil {
			return mail.Summary{}, expired(err)
		}
		rows, err := app.backend.Search(bounded, query)
		if err != nil {
			return mail.Summary{}, expired(err)
		}
		for _, row := range rows {
			if _, old := seen[row.ID]; old || (row.Received != nil && row.Received.Before(earliest)) {
				continue
			}
			return row, nil
		}
		select {
		case <-bounded.Done():
			return mail.Summary{}, expired(bounded.Err())
		case <-ticker.C:
		}
	}
}

// extract reads the message that arrived and finds its one-time code.
func (app *application) extract(ctx context.Context, location mail.Location, found mail.Summary) (string, error) {
	target := mail.Target{Location: location, Reference: mail.Reference{ID: found.ID, MessageID: found.MessageID}}
	message, err := app.backend.Read(ctx, target)
	if err != nil {
		return "", err
	}
	code, ok := mail.OneTimeCode(message.Subject + "\n" + message.Content)
	if !ok {
		return "", failure.New(failure.NotFound, fmt.Sprintf("message %d arrived but holds no one-time code", found.ID), fmt.Sprintf("read it with `mailslot read %d`", found.ID))
	}
	return code, nil
}
