package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ChiragAgg5k/mailslot/internal/failure"
	"github.com/ChiragAgg5k/mailslot/internal/mail"
)

const (
	toName      = "to"
	specialHint = "name the mailbox with `mailslot move --to <mailbox>`"
)

// destination picks the mailbox a relocating command moves messages into.
type destination func(ctx context.Context, account mail.Account) (string, error)

// refusal explains why a message already in the destination mailbox is left where it is.
type refusal func(mailbox string) error

func (app *application) move() *cobra.Command {
	var (
		mailbox string
		to      string
		dryRun  bool
	)
	command := &cobra.Command{
		Use:   "move <reference>... --to <mailbox>",
		Short: "Move messages to another mailbox of the same account",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(command *cobra.Command, references []string) error {
			if strings.TrimSpace(to) == "" {
				return failure.New(failure.Usage, "--to needs a mailbox name", usageHint(command))
			}
			named := func(context.Context, mail.Account) (string, error) {
				return to, nil
			}
			same := func(mailbox string) error {
				return failure.New(failure.Usage, fmt.Sprintf("the message is already in %s", mailbox), "pick another mailbox with --to")
			}
			return app.relocate(command, references, dryRun, named, same)
		},
	}
	mailboxFlag(command, &mailbox)
	command.Flags().StringVar(&to, toName, "", "mailbox to move the messages to")
	_ = command.MarkFlagRequired(toName)
	dryRunFlag(command, &dryRun)
	return changing(command)
}

// relocate moves messages into the mailbox that pick chooses for the account. A message already there
// is refused before anything changes, wherever its reference found it.
func (app *application) relocate(command *cobra.Command, references []string, dryRun bool, pick destination, refuse refusal) error {
	ctx := command.Context()
	account, err := app.resolve(ctx)
	if err != nil {
		return err
	}
	target, err := pick(ctx, account)
	if err != nil {
		return err
	}
	move := func(ctx context.Context, message mail.Target) (mail.Summary, error) {
		return app.backend.Move(ctx, message, target)
	}
	arrived := func(message mail.Target) error {
		if strings.EqualFold(message.Mailbox, target) {
			return refuse(target)
		}
		return nil
	}
	return app.apply(command, references, dryRun, "moved to "+target, move, true, arrived)
}

// special finds the account's mailbox with the given role.
func (app *application) special(role mail.Special) destination {
	return func(ctx context.Context, account mail.Account) (string, error) {
		mailboxes, err := app.backend.Mailboxes(ctx, account.Name)
		if err != nil {
			return "", err
		}
		name, found := role.Find(mailboxes)
		if !found {
			return "", failure.New(failure.NotFound, fmt.Sprintf("found no %s mailbox in account %q", strings.ToLower(string(role)), account.Name), specialHint)
		}
		return name, nil
	}
}
