package cli

import (
	"github.com/spf13/cobra"

	"github.com/ChiragAgg5k/mailslot/internal/failure"
)

func (app *application) archive() *cobra.Command {
	command := &cobra.Command{
		Use:   "archive <reference>...",
		Short: "Move messages to the account's archive mailbox",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(*cobra.Command, []string) error {
			return failure.New(failure.General, "not implemented", "")
		},
	}
	return changing(command)
}
