package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/ChiragAgg5k/amail/internal/mail"
)

const standardInput = "-"

// composition holds the flags shared by every command that writes a message.
type composition struct {
	body        string
	bodyFile    string
	attachments []string
	deliver     bool
}

func (composition *composition) flags(command *cobra.Command) {
	command.Flags().StringVar(&composition.body, "body", "", "message text")
	command.Flags().StringVar(&composition.bodyFile, "body-file", "", "file holding the message text (- for standard input)")
	command.Flags().StringArrayVar(&composition.attachments, "attach", nil, "file to attach (repeatable)")
	command.Flags().BoolVar(&composition.deliver, "send", false, "send immediately instead of opening a draft in Mail")
	command.MarkFlagsMutuallyExclusive("body", "body-file")
}

func (composition *composition) text(input io.Reader) (string, error) {
	switch composition.bodyFile {
	case "":
		if composition.body == "" {
			return "", errors.New("give the message text with --body or --body-file")
		}
		return composition.body, nil
	case standardInput:
		content, err := io.ReadAll(input)
		return string(content), err
	}
	content, err := os.ReadFile(composition.bodyFile)
	return string(content), err
}

func (composition *composition) files() ([]string, error) {
	paths := make([]string, 0, len(composition.attachments))
	for _, attachment := range composition.attachments {
		path, err := filepath.Abs(attachment)
		if err != nil {
			return nil, err
		}
		details, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("attachment: %w", err)
		}
		if details.IsDir() {
			return nil, fmt.Errorf("attachment %s is a directory", attachment)
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func (app *application) compose(command *cobra.Command, draft mail.Draft) error {
	delivery, err := app.client.Compose(command.Context(), draft)
	if err != nil {
		return err
	}
	return app.print(command.OutOrStdout(), delivery, func(output io.Writer) {
		if delivery.Sent {
			fmt.Fprintf(output, "Sent from %s.\n", delivery.Sender)
			return
		}
		fmt.Fprintln(output, "Opened as a draft in Mail; nothing was sent. Pass --send to send without review.")
	})
}

func (app *application) send() *cobra.Command {
	var composition composition
	draft := mail.Draft{}
	command := &cobra.Command{
		Use:   "send",
		Short: "Write a new message",
		Long:  "Write a new message. It opens as a draft in Mail unless --send is given.",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			body, err := composition.text(command.InOrStdin())
			if err != nil {
				return err
			}
			files, err := composition.files()
			if err != nil {
				return err
			}
			account, err := app.resolve(command.Context())
			if err != nil {
				return err
			}
			draft.Account = account.Name
			draft.Body = body
			draft.Attachments = files
			draft.Send = composition.deliver
			return app.compose(command, draft)
		},
	}
	command.Flags().StringArrayVar(&draft.To, "to", nil, "recipient (repeatable)")
	command.Flags().StringArrayVar(&draft.Cc, "cc", nil, "copied recipient (repeatable)")
	command.Flags().StringArrayVar(&draft.Bcc, "bcc", nil, "blind-copied recipient (repeatable)")
	command.Flags().StringVar(&draft.Subject, "subject", "", "subject line")
	_ = command.MarkFlagRequired("to")
	_ = command.MarkFlagRequired("subject")
	composition.flags(command)
	return command
}
