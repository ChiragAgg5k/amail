package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func (app *application) attachments() *cobra.Command {
	var mailbox, directory string
	command := &cobra.Command{
		Use:   "attachments <id>",
		Short: "Save the attachments of a message",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, arguments []string) error {
			_, target, err := app.target(command, arguments[0])
			if err != nil {
				return err
			}
			path, err := filepath.Abs(directory)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(path, 0o755); err != nil {
				return err
			}
			saved, err := app.backend.SaveAttachments(command.Context(), target, path)
			if err != nil {
				return err
			}
			return app.print(command.OutOrStdout(), saved, func(output io.Writer) {
				if len(saved) == 0 {
					fmt.Fprintln(output, "The message has no attachments.")
					return
				}
				for _, attachment := range saved {
					fmt.Fprintln(output, attachment.Path)
				}
			})
		},
	}
	mailboxFlag(command, &mailbox)
	command.Flags().StringVarP(&directory, "output", "o", ".", "directory to save into")
	return command
}
