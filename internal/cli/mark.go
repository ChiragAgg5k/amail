package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/ChiragAgg5k/mailslot/internal/mail"
)

const (
	readName   = "read"
	unreadName = "unread"
)

func (app *application) mark() *cobra.Command {
	var (
		mailbox string
		read    bool
		unread  bool
		dryRun  bool
	)
	command := &cobra.Command{
		Use:   "mark <reference>... (--read | --unread)",
		Short: "Mark messages as read or unread",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(command *cobra.Command, references []string) error {
			describe := "marked unread"
			if read {
				describe = "marked read"
			}
			return app.apply(command, references, dryRun, describe, func(ctx context.Context, target mail.Target) (mail.Summary, error) {
				return app.backend.Mark(ctx, target, read)
			}, false)
		},
	}
	mailboxFlag(command, &mailbox)
	command.Flags().BoolVar(&read, readName, false, "mark the messages as read")
	command.Flags().BoolVar(&unread, unreadName, false, "mark the messages as unread")
	command.MarkFlagsMutuallyExclusive(readName, unreadName)
	command.MarkFlagsOneRequired(readName, unreadName)
	dryRunFlag(command, &dryRun)
	return changing(command)
}
