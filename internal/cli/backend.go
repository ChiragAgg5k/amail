package cli

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/ChiragAgg5k/mailslot/internal/failure"
	"github.com/ChiragAgg5k/mailslot/internal/fixture"
	"github.com/ChiragAgg5k/mailslot/internal/mail"
)

const (
	fixtureVariable      = "MAILSLOT_FIXTURE"
	fixtureStateVariable = "MAILSLOT_FIXTURE_STATE"
	lockTimeoutVariable  = "MAILSLOT_LOCK_TIMEOUT"
)

// open picks the backend, Mail itself or the fixture named by MAILSLOT_FIXTURE, behind the queue that
// lets one mailslot command at a time through.
func (app *application) open() (mail.Backend, error) {
	var backend mail.Backend = mail.Client{}
	if app.fixture != "" {
		fixed, err := fixture.Open(app.fixture, os.Getenv(fixtureStateVariable))
		if err != nil {
			var typed *failure.Error
			if errors.As(err, &typed) {
				return nil, err
			}
			return nil, failure.Wrap(failure.General, fmt.Errorf("open fixture %s: %w", app.fixture, err), "")
		}
		backend = fixed
	}
	path, err := mail.LockPath()
	if err != nil {
		return nil, failure.Wrap(failure.General, err, "")
	}
	timeout := mail.DefaultTimeout
	if value := os.Getenv(lockTimeoutVariable); value != "" {
		parsed, err := time.ParseDuration(value)
		if err != nil || parsed <= 0 {
			return nil, failure.New(failure.Usage, fmt.Sprintf("%s=%q is not a positive duration", lockTimeoutVariable, value), "use a Go duration such as 30s or 2m")
		}
		timeout = parsed
	}
	return mail.Queue{Backend: backend, Path: path, Timeout: timeout}, nil
}
