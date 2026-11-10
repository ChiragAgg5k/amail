package mail

import "context"

// Source returns the raw RFC 822 source of a message.
func (client Client) Source(ctx context.Context, target Target) (string, error) {
	var response struct {
		Source string `json:"source"`
	}
	err := client.run(ctx, sourceScript, target, target, &response)
	return response.Source, err
}
