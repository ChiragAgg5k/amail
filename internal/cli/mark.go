package cli

import (
	"github.com/spf13/cobra"

	"github.com/ChiragAgg5k/mailslot/internal/failure"
)

func (app *application) mark() *cobra.Command {
	command := &cobra.Command{
		Use:   "mark <reference>...",
		Short: "Mark messages as read or unread",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(*cobra.Command, []string) error {
			return failure.New(failure.General, "not implemented", "")
		},
	}
	return changing(command)
}
