package mail

import "context"

// Backend is everything mailslot asks of Mail. The JXA Client talks to the real app; tests use a fixture.
type Backend interface {
	// Ping checks that Mail answers a real Apple event.
	Ping(ctx context.Context) (Status, error)
	// Accounts lists the real accounts, never the local pseudo-account.
	Accounts(ctx context.Context) ([]Account, error)
	Mailboxes(ctx context.Context, account string) ([]Mailbox, error)
	List(ctx context.Context, location Location, limit int, unread bool) ([]Summary, error)
	Search(ctx context.Context, query Query) ([]Summary, error)
	// Resolve finds the message a target names and reports the mailbox it was found in.
	Resolve(ctx context.Context, target Target) (Summary, error)
	Read(ctx context.Context, target Target) (Message, error)
	Source(ctx context.Context, target Target) (string, error)
	SaveAttachments(ctx context.Context, target Target, directory string) ([]SavedAttachment, error)
	Mark(ctx context.Context, target Target, read bool) (Summary, error)
	Flag(ctx context.Context, target Target, color FlagColor) (Summary, error)
	// Move returns the message at its destination, with ID 0 when Mail has not assigned the new id yet.
	Move(ctx context.Context, target Target, destination string) (Summary, error)
	Compose(ctx context.Context, draft Draft) (Delivery, error)
	// Fetch asks Mail to check for new mail.
	Fetch(ctx context.Context, account string) error
}
