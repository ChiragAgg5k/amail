package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/ChiragAgg5k/mailslot/internal/mail"
)

const (
	colorName = "color"
	clearName = "clear"
)

func (app *application) flag() *cobra.Command {
	var (
		mailbox  string
		name     string
		clearing bool
		dryRun   bool
	)
	command := &cobra.Command{
		Use:   "flag <reference>... (--color <name> | --clear)",
		Short: "Flag messages with a colour, or clear their flag",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(command *cobra.Command, references []string) error {
			color := mail.Clear
			describe := "flag cleared"
			if !clearing {
				parsed, err := mail.ParseFlagColor(name)
				if err != nil {
					return err
				}
				color = parsed
				describe = "flagged " + color.String()
			}
			return app.apply(command, references, dryRun, describe, func(ctx context.Context, target mail.Target) (mail.Summary, error) {
				return app.backend.Flag(ctx, target, color)
			}, false)
		},
	}
	mailboxFlag(command, &mailbox)
	command.Flags().StringVar(&name, colorName, "", "flag colour: red, orange, yellow, green, blue, purple or gray")
	command.Flags().BoolVar(&clearing, clearName, false, "remove the flag")
	command.MarkFlagsMutuallyExclusive(colorName, clearName)
	command.MarkFlagsOneRequired(colorName, clearName)
	dryRunFlag(command, &dryRun)
	return changing(command)
}
