package mail

import "context"

// Fetch asks Mail to check an account, or every account for the local pseudo-account, for new mail.
func (client Client) Fetch(ctx context.Context, account string) error {
	return client.run(ctx, checkScript, Target{}, struct {
		Account string `json:"account"`
	}{account}, nil)
}
