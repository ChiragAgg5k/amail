package cli

import (
	"github.com/spf13/cobra"

	"github.com/ChiragAgg5k/mailslot/internal/failure"
)

func (app *application) flag() *cobra.Command {
	command := &cobra.Command{
		Use:   "flag <reference>...",
		Short: "Flag messages with a colour, or clear their flag",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(*cobra.Command, []string) error {
			return failure.New(failure.General, "not implemented", "")
		},
	}
	return changing(command)
}
