package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ChiragAgg5k/mailslot/internal/failure"
	"github.com/ChiragAgg5k/mailslot/internal/mail"
)

const (
	rawName         = "raw"
	outputName      = "output"
	readDescription = `Show one message in full: its headers, attachment names and text.

With --raw, print the message's RFC 822 source exactly as Mail holds it, or save it as an .eml file
with --output. On Exchange accounts the source may be incomplete when Mail has not downloaded the
whole message.`
)

// rawSource is the JSON form of read --raw. Source is left out when the source was saved to Path.
type rawSource struct {
	ID        int    `json:"id"`
	MessageID string `json:"messageId"`
	Mailbox   string `json:"mailbox"`
	Source    string `json:"source,omitempty"`
	Path      string `json:"path,omitempty"`
}

func (app *application) read() *cobra.Command {
	var mailbox, output string
	var raw bool
	command := &cobra.Command{
		Use:   "read <id>",
		Short: "Show one message in full",
		Long:  readDescription,
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, arguments []string) error {
			if output != "" && !raw {
				return failure.New(failure.Usage, fmt.Sprintf("--%s only works with --%s", outputName, rawName), usageHint(command))
			}
			_, target, err := app.target(command, arguments[0])
			if err != nil {
				return err
			}
			if raw {
				return app.readSource(command, target, output)
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
	command.Flags().BoolVar(&raw, rawName, false, "print the raw RFC 822 source (may be incomplete on Exchange)")
	command.Flags().StringVarP(&output, outputName, "o", "", "with --raw, save the source to this .eml file instead of printing it")
	return command
}

// readSource prints or saves the raw source of a message.
func (app *application) readSource(command *cobra.Command, target mail.Target, path string) error {
	source, err := app.backend.Source(command.Context(), target)
	if err != nil {
		return err
	}
	if path != "" {
		absolute, err := filepath.Abs(path)
		if err != nil {
			return failure.Wrap(failure.General, fmt.Errorf("locate %s: %w", path, err), "")
		}
		if err := os.WriteFile(absolute, []byte(source), 0o600); err != nil {
			return failure.Wrap(failure.General, fmt.Errorf("save message source: %w", err), "check that the directory exists and is writable")
		}
		path = absolute
	}
	if !app.json {
		if path != "" {
			_, err = fmt.Fprintln(command.OutOrStdout(), path)
			return err
		}
		_, err = io.WriteString(command.OutOrStdout(), source)
		return err
	}
	found, err := app.backend.Resolve(command.Context(), target)
	if err != nil {
		return err
	}
	described := rawSource{ID: found.ID, MessageID: found.MessageID, Mailbox: found.Mailbox, Path: path}
	if path == "" {
		described.Source = source
	}
	return app.print(command.OutOrStdout(), described, func(io.Writer) {})
}
