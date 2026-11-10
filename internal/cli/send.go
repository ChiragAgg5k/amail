package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/ChiragAgg5k/mailslot/internal/failure"
	"github.com/ChiragAgg5k/mailslot/internal/mail"
)

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
	command.Flags().StringArrayVar(&draft.To, toName, nil, "recipient (repeatable)")
	command.Flags().StringArrayVar(&draft.Cc, ccName, nil, "copied recipient (repeatable)")
	command.Flags().StringArrayVar(&draft.Bcc, bccName, nil, "blind-copied recipient (repeatable)")
	command.Flags().StringVar(&draft.Subject, subjectName, "", "subject line")
	_ = command.MarkFlagRequired(toName)
	_ = command.MarkFlagRequired(subjectName)
	composition.flags(command)
	return changing(command)
}
