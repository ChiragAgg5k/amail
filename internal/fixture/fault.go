package fixture

import (
	"fmt"

	"github.com/ChiragAgg5k/mailslot/internal/failure"
	"github.com/ChiragAgg5k/mailslot/internal/mail"
)

// FaultVariable names the environment variable that makes every fixture call fail the way Mail can.
const FaultVariable = "MAILSLOT_FIXTURE_FAULT"

// Fault is a Mail failure the fixture backend imitates on every call.
type Fault string

const (
	NoFault    Fault = ""
	Permission Fault = "permission"
	Timeout    Fault = "timeout"
)

// ParseFault reads the value of FaultVariable.
func ParseFault(value string) (Fault, error) {
	switch fault := Fault(value); fault {
	case NoFault, Permission, Timeout:
		return fault, nil
	}
	return NoFault, failure.New(failure.General, fmt.Sprintf("unknown %s value %q", FaultVariable, value), fmt.Sprintf("use %q or %q, or leave it unset", Permission, Timeout))
}

func (fault Fault) err() error {
	switch fault {
	case Permission:
		return mail.PermissionDenied()
	case Timeout:
		return mail.Unanswered(mail.DefaultTimeout)
	}
	return nil
}
