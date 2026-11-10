package cli

import (
	"github.com/spf13/cobra"
)

func (app *application) reply() *cobra.Command {
	var composition composition
	var mailbox string
	var everyone bool
	command := &cobra.Command{
		Use:   "reply <id>",
		Short: "Answer a message",
		Long: "Answer a message with the original quoted below your text. It opens as a draft in Mail unless --send is given.\n\n" +
			"The reply is a new message that threads by its Re: subject; it carries no In-Reply-To header.",
		Args: cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, arguments []string) error {
			body, err := composition.text(command.InOrStdin())
			if err != nil {
				return err
			}
			files, err := composition.files()
			if err != nil {
				return err
			}
			account, target, err := app.target(command, arguments[0])
			if err != nil {
				return err
			}
			if err := outgoing(account); err != nil {
				return err
			}
			if err := composition.sender(account); err != nil {
				return err
			}
			original, err := app.backend.Read(command.Context(), target)
			if err != nil {
				return err
			}
			draft := original.Reply(body, everyone, account.Addresses)
			composition.sign(&draft, account)
			draft.Attachments = files
			return app.compose(command, draft)
		},
	}
	mailboxFlag(command, &mailbox)
	command.Flags().BoolVar(&everyone, "all", false, "copy the other recipients of the original")
	composition.flags(command)
	return changing(command)
}
