package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

func (app *application) read() *cobra.Command {
	var mailbox string
	command := &cobra.Command{
		Use:   "read <id>",
		Short: "Show one message in full",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, arguments []string) error {
			_, target, err := app.target(command, arguments[0])
			if err != nil {
				return err
			}
			message, err := app.backend.Read(command.Context(), target)
			if err != nil {
				return err
			}
			return app.print(command.OutOrStdout(), message, func(output io.Writer) {
				fmt.Fprintf(output, "From:    %s\n", message.Sender)
				fmt.Fprintf(output, "To:      %s\n", recipients(message.To))
				if len(message.Cc) > 0 {
					fmt.Fprintf(output, "Cc:      %s\n", recipients(message.Cc))
				}
				fmt.Fprintf(output, "Date:    %s\n", local(message.Received))
				fmt.Fprintf(output, "Subject: %s\n", message.Subject)
				if len(message.Attachments) > 0 {
					names := make([]string, 0, len(message.Attachments))
					for _, attachment := range message.Attachments {
						names = append(names, attachment.Name)
					}
					fmt.Fprintf(output, "Files:   %s\n", strings.Join(names, ", "))
				}
				fmt.Fprintf(output, "\n%s\n", strings.TrimSpace(message.Content))
			})
		},
	}
	mailboxFlag(command, &mailbox)
	return command
}
