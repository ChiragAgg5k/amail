package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
)

func (app *application) attachments() *cobra.Command {
	var mailbox, directory string
	command := &cobra.Command{
		Use:   "attachments <id>",
		Short: "Save the attachments of a message",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, arguments []string) error {
			id, err := strconv.Atoi(arguments[0])
			if err != nil {
				return notIdentifier(arguments[0])
			}
			target, err := filepath.Abs(directory)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			_, location, err := app.locate(command.Context(), mailbox)
			if err != nil {
				return err
			}
			saved, err := app.client.SaveAttachments(command.Context(), location, id, target)
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

func notIdentifier(value string) error {
	return fmt.Errorf("%q is not a message id; take one from `amail list`", value)
}
