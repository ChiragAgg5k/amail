package cli

import (
	"github.com/spf13/cobra"

	"github.com/ChiragAgg5k/mailslot/internal/mail"
)

// target turns a message reference from the command line into the account and message it names.
// A Message-ID without an explicit --mailbox is looked for in every mailbox of the account.
func (app *application) target(command *cobra.Command, value string) (mail.Account, mail.Target, error) {
	reference, err := mail.ParseReference(value)
	if err != nil {
		return mail.Account{}, mail.Target{}, err
	}
	account, err := app.resolve(command.Context())
	if err != nil {
		return mail.Account{}, mail.Target{}, err
	}
	return account, locate(command, account, reference), nil
}

func locate(command *cobra.Command, account mail.Account, reference mail.Reference) mail.Target {
	mailbox := mail.Inbox
	if flag := command.Flags().Lookup(mailboxName); flag != nil {
		mailbox = flag.Value.String()
		if reference.MessageID != "" && !flag.Changed {
			mailbox = ""
		}
	}
	return mail.Target{Location: mail.Location{Account: account.Name, Mailbox: mailbox}, Reference: reference}
}
