package cli

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ChiragAgg5k/mailslot/internal/mail"
)

const (
	dryRunNotice = "Dry run; nothing changed."
	unknownID    = "unknown"
)

// change is one row of a triage command's report.
type change struct {
	ID        int    `json:"id"`
	MessageID string `json:"messageId"`
	Mailbox   string `json:"mailbox"`
	Subject   string `json:"subject"`
	Change    string `json:"change"`
	// NewID is the id after a move; 0 when Mail has not assigned it yet, absent for changes that do not move.
	NewID  *int `json:"newId,omitempty"`
	DryRun bool `json:"dryRun"`
}

// action changes one resolved message and returns it as it is afterwards.
type action func(ctx context.Context, target mail.Target) (mail.Summary, error)

// guard refuses a resolved message before anything changes; a nil guard accepts every message.
type guard func(target mail.Target) error

// resolved identifies one message after its reference is resolved, so two references to it collapse.
// It holds the lowercased Message-ID when the message has one, else the lowercased mailbox and id,
// so Inbox and INBOX name the same message.
type resolved struct {
	messageID string
	mailbox   string
	id        int
}

func identity(target mail.Target) resolved {
	if target.MessageID != "" {
		return resolved{messageID: strings.ToLower(target.MessageID)}
	}
	return resolved{mailbox: strings.ToLower(target.Mailbox), id: target.ID}
}

// apply runs a triage change over message references. Every reference is resolved and guarded before
// anything changes, so a bad or refused reference changes nothing; references to the same message
// count once. After that, a failure reports the rows already changed.
func (app *application) apply(command *cobra.Command, references []string, dryRun bool, describe string, act action, moves bool, refuse guard) error {
	ctx := command.Context()
	parsed := make([]mail.Reference, 0, len(references))
	for _, value := range references {
		reference, err := mail.ParseReference(value)
		if err != nil {
			return err
		}
		parsed = append(parsed, reference)
	}
	account, err := app.resolve(ctx)
	if err != nil {
		return err
	}

	targets := make([]mail.Target, 0, len(parsed))
	rows := make([]change, 0, len(parsed))
	seen := make(map[resolved]struct{}, len(parsed))
	for _, reference := range parsed {
		target := locate(command, account, reference)
		found, err := app.backend.Resolve(ctx, target)
		if err != nil {
			return err
		}
		if found.Mailbox != "" {
			target.Mailbox = found.Mailbox
		}
		target.ID = found.ID
		if found.MessageID != "" {
			target.MessageID = found.MessageID
		}
		key := identity(target)
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		if refuse != nil {
			if err := refuse(target); err != nil {
				return err
			}
		}
		targets = append(targets, target)
		rows = append(rows, change{
			ID:        found.ID,
			MessageID: target.MessageID,
			Mailbox:   target.Mailbox,
			Subject:   found.Subject,
			Change:    describe,
			DryRun:    dryRun,
		})
	}

	output := command.OutOrStdout()
	if dryRun {
		return app.changed(output, rows, true)
	}
	for index, target := range targets {
		after, err := act(ctx, target)
		if err != nil {
			if printed := app.changed(output, rows[:index], false); printed != nil {
				return printed
			}
			return err
		}
		if moves {
			id := after.ID
			rows[index].NewID = &id
		}
	}
	return app.changed(output, rows, false)
}

func (app *application) changed(output io.Writer, rows []change, dryRun bool) error {
	return app.print(output, rows, func(output io.Writer) {
		if dryRun {
			fmt.Fprintln(output, dryRunNotice)
		}
		if len(rows) == 0 {
			return
		}
		columns(output, "ID\tMESSAGE-ID\tCHANGE\tNEW ID\tSUBJECT", func(writer io.Writer) {
			for _, row := range rows {
				fmt.Fprintf(writer, "%d\t%s\t%s\t%s\t%s\n", row.ID, row.MessageID, row.Change, newID(row.NewID), row.Subject)
			}
		})
	})
}

func newID(id *int) string {
	switch {
	case id == nil:
		return ""
	case *id == 0:
		return unknownID
	}
	return strconv.Itoa(*id)
}

func dryRunFlag(command *cobra.Command, target *bool) {
	command.Flags().BoolVar(target, dryRunName, false, "show what would change without changing anything")
}
