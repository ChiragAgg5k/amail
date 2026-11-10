package mail

import "context"

// Status is what Mail reports when it answers a ping.
type Status struct {
	Version string `json:"version"`
	// Accounts comes from accounts.length, a real Apple event; asking only for the version never reaches Mail.
	Accounts int `json:"accounts"`
}

// Ping checks that Mail is present, running and permitted to answer.
func (client Client) Ping(ctx context.Context) (Status, error) {
	var status Status
	err := client.run(ctx, pingScript, Target{}, nil, &status)
	return status, err
}
