package mail

import "context"

// Attachment describes a file attached to a message.
type Attachment struct {
	Name string `json:"name"`
	Size int    `json:"size"`
	Type string `json:"type"`
}

// SavedAttachment is an attachment written to disk.
type SavedAttachment struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// SaveAttachments writes every attachment of a message into a directory.
func (client Client) SaveAttachments(ctx context.Context, target Target, directory string) ([]SavedAttachment, error) {
	var saved []SavedAttachment
	err := client.run(ctx, attachmentsScript, target, struct {
		Target
		Directory string `json:"directory"`
	}{target, directory}, &saved)
	return saved, err
}
