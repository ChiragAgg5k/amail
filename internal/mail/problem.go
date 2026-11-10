package mail

import (
	"fmt"
	"time"

	"github.com/ChiragAgg5k/mailslot/internal/failure"
)

// PermissionDenied reports that macOS refused this process control of Mail.
func PermissionDenied() *failure.Error {
	return failure.New(
		failure.Permission,
		"the app running this command is not allowed to control Mail",
		"allow it under System Settings → Privacy & Security → Automation; some apps (for example agent hosts without the Apple Events entitlement) cannot be granted this permission at all, so run mailslot from a terminal app instead",
	)
}

// Unanswered reports that Mail gave no reply within the timeout.
func Unanswered(timeout time.Duration) *failure.Error {
	return failure.New(
		failure.Busy,
		fmt.Sprintf("Mail did not answer within %s", timeout),
		"Mail may be busy, or a permission prompt may be waiting unanswered on screen; check for it and try again",
	)
}

// MaybeSent reports that Mail did not answer a send in time. Stopping osascript does not cancel the Apple event,
// so Mail may still send the message, and retrying could send it twice.
func MaybeSent(problem *failure.Error) *failure.Error {
	return &failure.Error{
		Code:    problem.Code,
		Message: problem.Message,
		Hint:    "Mail may still have sent it; check the Sent mailbox before retrying",
		Cause:   problem.Cause,
	}
}
