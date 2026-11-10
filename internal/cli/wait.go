package cli

import (
	"github.com/spf13/cobra"

	"github.com/ChiragAgg5k/mailslot/internal/failure"
)

func (app *application) wait() *cobra.Command {
	command := &cobra.Command{
		Use:   "wait",
		Short: "Wait for a new message from a sender or with a subject to arrive",
		Args:  cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			return failure.New(failure.General, "not implemented", "")
		},
	}
	return command
}
