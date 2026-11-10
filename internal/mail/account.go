package mail

import (
	"context"
	"strings"
)

// Account is a mail account configured in Mail.
type Account struct {
	Name      string   `json:"name"`
	Addresses []string `json:"addresses"`
	Kind      string   `json:"kind"`
	Enabled   bool     `json:"enabled"`
}

// Owns reports whether an address belongs to the account, ignoring case.
func (account Account) Owns(address string) bool {
	for _, own := range account.Addresses {
		if strings.EqualFold(own, strings.TrimSpace(address)) {
			return true
		}
	}
	return false
}

// Accounts lists every account configured in Mail.
func (client Client) Accounts(ctx context.Context) ([]Account, error) {
	var accounts []Account
	err := client.run(ctx, accountsScript, Target{}, nil, &accounts)
	return accounts, err
}
