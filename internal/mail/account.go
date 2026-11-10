package mail

import "context"

// Account is a mail account configured in Mail.
type Account struct {
	Name      string   `json:"name"`
	Addresses []string `json:"addresses"`
	Kind      string   `json:"kind"`
	Enabled   bool     `json:"enabled"`
}

// Accounts lists every account configured in Mail.
func (client Client) Accounts(ctx context.Context) ([]Account, error) {
	var accounts []Account
	err := client.run(ctx, "accounts", nil, &accounts)
	return accounts, err
}
