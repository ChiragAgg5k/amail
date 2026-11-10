package cli

import (
	"github.com/spf13/cobra"

	"github.com/ChiragAgg5k/mailslot/internal/failure"
)

func (app *application) delete() *cobra.Command {
	command := &cobra.Command{
		Use:   "delete <reference>...",
		Short: "Move messages to the account's Trash; nothing is erased",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(*cobra.Command, []string) error {
			return failure.New(failure.General, "not implemented", "")
		},
	}
	return changing(command)
}
