// Package failure gives errors an exit code and a hint for the person reading them.
package failure

// Code is the process exit code a failure maps to.
type Code int

const (
	// Success means the command did what was asked.
	Success Code = 0
	// General is any failure without a more specific code.
	General Code = 1
	// Usage means the command line was wrong.
	Usage Code = 2
	// Permission means macOS did not allow this process to control Mail.
	Permission Code = 3
	// NotFound means a message, mailbox, account or signature does not exist.
	NotFound Code = 4
	// Busy means Mail or another mailslot command did not answer in time.
	Busy Code = 5
	// Refused means a safety switch or check stopped the command.
	Refused Code = 6
)
