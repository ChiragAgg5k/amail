package cli

import (
	"github.com/spf13/cobra"

	"github.com/ChiragAgg5k/mailslot/internal/failure"
)

func (app *application) merge() *cobra.Command {
	command := &cobra.Command{
		Use:   "merge <csv>",
		Short: "Open one draft per row of a CSV file",
		Args:  cobra.ExactArgs(1),
		RunE: func(*cobra.Command, []string) error {
			return failure.New(failure.General, "not implemented", "")
		},
	}
	return changing(command)
}
