package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ChiragAgg5k/mailslot/internal/config"
	"github.com/ChiragAgg5k/mailslot/internal/failure"
)

func (app *application) accountsCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "accounts",
		Short: "List the accounts configured in Mail",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			accounts, err := app.accounts(command.Context())
			if err != nil {
				return err
			}
			return app.print(command.OutOrStdout(), accounts, func(output io.Writer) {
				columns(output, "NAME\tADDRESSES\tKIND\tENABLED", func(writer io.Writer) {
					for _, account := range accounts {
						fmt.Fprintf(writer, "%s\t%s\t%s\t%t\n", account.Name, strings.Join(account.Addresses, ", "), account.Kind, account.Enabled)
					}
				})
			})
		},
	}
	command.AddCommand(&cobra.Command{
		Use:   "use <name>",
		Short: "Make an account the default",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, arguments []string) error {
			app.account = arguments[0]
			account, err := app.resolve(command.Context())
			if err != nil {
				return err
			}
			stored, err := config.Load()
			if err != nil {
				return failure.Wrap(failure.General, fmt.Errorf("read configuration: %w", err), "")
			}
			stored.Account = account.Name
			if err := stored.Save(); err != nil {
				return failure.Wrap(failure.General, fmt.Errorf("write configuration: %w", err), "")
			}
			fmt.Fprintf(command.OutOrStdout(), "Default account is now %s.\n", account.Name)
			return nil
		},
	})
	return command
}
