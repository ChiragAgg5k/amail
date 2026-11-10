package cli

import (
	"fmt"
	"io"
	netmail "net/mail"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ChiragAgg5k/mailslot/internal/failure"
	"github.com/ChiragAgg5k/mailslot/internal/mail"
)

const standardInput = "-"

// composition holds the flags shared by every command that writes a message.
type composition struct {
	body        string
	bodyFile    string
	attachments []string
	from        string
	signature   string
	deliver     bool
}

func (composition *composition) flags(command *cobra.Command) {
	command.Flags().StringVar(&composition.body, "body", "", "message text")
	command.Flags().StringVar(&composition.bodyFile, "body-file", "", "file holding the message text (- for standard input)")
	command.Flags().StringArrayVar(&composition.attachments, "attach", nil, "file to attach (repeatable)")
	command.Flags().StringVar(&composition.from, "from", "", "sender address; must belong to the account (default: the account's first address)")
	command.Flags().StringVar(&composition.signature, "signature", "", "name of a Mail signature to append (default: none)")
	command.Flags().BoolVar(&composition.deliver, sendName, false, "send immediately instead of opening a draft in Mail")
	command.MarkFlagsMutuallyExclusive("body", "body-file")
}

func (composition *composition) text(input io.Reader) (string, error) {
	switch composition.bodyFile {
	case "":
		if composition.body == "" {
			return "", failure.New(failure.Usage, "give the message text with --body or --body-file", "")
		}
		return composition.body, nil
	case standardInput:
		content, err := io.ReadAll(input)
		if err != nil {
			return "", fmt.Errorf("read message text from standard input: %w", err)
		}
		return string(content), nil
	}
	content, err := os.ReadFile(composition.bodyFile)
	if err != nil {
		return "", failure.Wrap(failure.Usage, fmt.Errorf("body file: %w", err), "")
	}
	return string(content), nil
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
			return nil, failure.Wrap(failure.Usage, fmt.Errorf("attachment: %w", err), "")
		}
		if details.IsDir() {
			return nil, failure.New(failure.Usage, fmt.Sprintf("attachment %s is a directory", attachment), "")
		}
		paths = append(paths, path)
	}
	return paths, nil
}

// sender refuses a --from address the account does not own, before Mail is asked to compose anything.
func (composition *composition) sender(account mail.Account) error {
	if composition.from == "" {
		return nil
	}
	address := composition.from
	if parsed, err := netmail.ParseAddress(composition.from); err == nil {
		address = parsed.Address
	}
	if account.Owns(address) {
		return nil
	}
	return failure.New(
		failure.Usage,
		fmt.Sprintf("%s is not an address of account %q", composition.from, account.Name),
		"pick one of: "+strings.Join(account.Addresses, ", "),
	)
}

// sign copies the account, sender, signature and send choice onto the draft.
func (composition *composition) sign(draft *mail.Draft, account mail.Account) {
	draft.Account = account.Name
	draft.From = composition.from
	draft.Signature = composition.signature
	draft.Send = composition.deliver
}

// outgoing refuses the local pseudo-account, which has no server to send through.
func outgoing(account mail.Account) error {
	if account.Name == mail.Local {
		return failure.New(failure.Usage, "local mailboxes cannot send mail; pick an account with --account", "")
	}
	return nil
}

// compose hands the draft to Mail, which reads the sender back and refuses a mismatch before sending.
func (app *application) compose(command *cobra.Command, draft mail.Draft) error {
	delivery, err := app.backend.Compose(command.Context(), draft)
	if err != nil {
		return err
	}
	return app.print(command.OutOrStdout(), delivery, func(output io.Writer) {
		if delivery.Sent {
			fmt.Fprintf(output, "Sent from %s.\n", delivery.Sender)
			return
		}
		fmt.Fprintf(output, "Opened as a draft from %s in Mail; nothing was sent. Pass --send to send without review.\n", delivery.Sender)
	})
}

func (app *application) send() *cobra.Command {
	var composition composition
	draft := mail.Draft{}
	command := &cobra.Command{
		Use:   "send",
		Short: "Write a new message",
		Long:  "Write a new message. It opens as a draft in Mail unless --send is given. Mail's sender is read back and checked before anything is sent.",
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
			if err := outgoing(account); err != nil {
				return err
			}
			if err := composition.sender(account); err != nil {
				return err
			}
			composition.sign(&draft, account)
			draft.Body = body
			draft.Attachments = files
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
	return changing(command)
}
