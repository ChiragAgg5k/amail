package mail

const (
	// Local is the reserved pseudo-account name for the "On My Mac" mailboxes.
	Local = "local"
	// LocalKind is the kind reported for the local pseudo-account.
	LocalKind = "local"
)

// LocalAccount describes the "On My Mac" mailboxes as an account.
func LocalAccount() Account {
	return Account{Name: Local, Kind: LocalKind, Addresses: []string{}, Enabled: true}
}
