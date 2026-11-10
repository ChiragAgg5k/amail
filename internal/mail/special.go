package mail

import "strings"

// Special is a mailbox with a role that every account names its own way.
//
// The candidate names follow what common providers call these mailboxes (IMAP servers, Gmail, iCloud,
// Exchange and Mail's On My Mac) and need confirming per provider in the e2e suite.
type Special string

const (
	Archive Special = "Archive"
	Trash   Special = "Trash"
)

// Find picks the mailbox that plays this role, matching each candidate name in order against the
// mailbox names and then their leaf after the last "/", ignoring case.
func (special Special) Find(mailboxes []Mailbox) (string, bool) {
	for _, candidate := range special.names() {
		for _, mailbox := range mailboxes {
			if strings.EqualFold(mailbox.Name, candidate) {
				return mailbox.Name, true
			}
		}
		for _, mailbox := range mailboxes {
			leaf := mailbox.Name[strings.LastIndex(mailbox.Name, "/")+1:]
			if strings.EqualFold(leaf, candidate) {
				return mailbox.Name, true
			}
		}
	}
	return "", false
}

func (special Special) names() []string {
	switch special {
	case Archive:
		return []string{"Archive", "Archives", "All Mail"}
	case Trash:
		return []string{"Trash", "Deleted Items", "Deleted Messages", "Bin"}
	}
	return nil
}
