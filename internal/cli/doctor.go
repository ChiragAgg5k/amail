package cli

import (
	"github.com/spf13/cobra"

	"github.com/ChiragAgg5k/mailslot/internal/failure"
)

func (app *application) doctor() *cobra.Command {
	command := &cobra.Command{
		Use:   "doctor",
		Short: "Check that Mail is present, answering and permitted, without changing anything",
		Args:  cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			return failure.New(failure.General, "not implemented", "")
		},
	}
	return command
}
