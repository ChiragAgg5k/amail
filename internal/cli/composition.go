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

const (
	standardInput = "-"
	bodyFileName  = "body-file"
	attachName    = "attach"
	fromName      = "from"
	signatureName = "signature"
	ccName        = "cc"
	bccName       = "bcc"
	subjectName   = "subject"
)

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
	command.Flags().StringVar(&composition.body, bodyName, "", "message text")
	command.Flags().StringVar(&composition.bodyFile, bodyFileName, "", "file holding the message text (- for standard input)")
	command.Flags().StringArrayVar(&composition.attachments, attachName, nil, "file to attach (repeatable)")
	command.Flags().StringVar(&composition.from, fromName, "", "sender address; must belong to the account (default: the account's first address)")
	command.Flags().StringVar(&composition.signature, signatureName, "", "name of a Mail signature to append (default: none)")
	command.Flags().BoolVar(&composition.deliver, sendName, false, "send immediately instead of opening a draft in Mail")
	command.MarkFlagsMutuallyExclusive(bodyName, bodyFileName)
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

// address is the bare address of --from, so "Sam Rivera <sam@example.com>" reaches Mail as sam@example.com.
func (composition *composition) address() string {
	if parsed, err := netmail.ParseAddress(composition.from); err == nil {
		return parsed.Address
	}
	return strings.TrimSpace(composition.from)
}

// sender refuses a --from address the account does not own, before Mail is asked to compose anything.
func (composition *composition) sender(account mail.Account) error {
	if composition.from == "" || account.Owns(composition.address()) {
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
	draft.From = composition.address()
	draft.Signature = composition.signature
	draft.Send = composition.deliver
}
