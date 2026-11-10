package mail

import "context"

// Inbox is the mailbox used when none is named. Names are matched without regard to case.
const Inbox = "Inbox"

// Mailbox is a folder of an account.
type Mailbox struct {
	Name     string `json:"name"`
	Unread   int    `json:"unread"`
	Messages int    `json:"messages"`
}

// Location points at one mailbox of one account.
type Location struct {
	Account string `json:"account"`
	Mailbox string `json:"mailbox"`
}

// Mailboxes lists the mailboxes of an account.
func (client Client) Mailboxes(ctx context.Context, account string) ([]Mailbox, error) {
	var mailboxes []Mailbox
	err := client.run(ctx, "mailboxes", struct {
		Account string `json:"account"`
	}{account}, &mailboxes)
	return mailboxes, err
}
