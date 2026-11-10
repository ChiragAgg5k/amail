package mail

import "context"

// Resolve finds the message a target names and reports where it was found.
func (client Client) Resolve(ctx context.Context, target Target) (Summary, error) {
	var summary Summary
	err := client.run(ctx, resolveScript, target, target, &summary)
	return summary, err
}

// Mark sets a message's read state.
func (client Client) Mark(ctx context.Context, target Target, read bool) (Summary, error) {
	var summary Summary
	err := client.run(ctx, markScript, target, struct {
		Target
		Read bool `json:"read"`
	}{target, read}, &summary)
	return summary, err
}

// Flag sets a message's flag colour, or clears the flag.
func (client Client) Flag(ctx context.Context, target Target, color FlagColor) (Summary, error) {
	var summary Summary
	err := client.run(ctx, flagScript, target, struct {
		Target
		Index int `json:"index"`
	}{target, int(color)}, &summary)
	return summary, err
}

// Move puts a message into another mailbox of the same account.
func (client Client) Move(ctx context.Context, target Target, destination string) (Summary, error) {
	var summary Summary
	err := client.run(ctx, moveScript, target, struct {
		Target
		Destination string `json:"destination"`
	}{target, destination}, &summary)
	return summary, err
}
