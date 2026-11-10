package cli

import (
	"context"
	"testing"
	"time"

	"github.com/ChiragAgg5k/mailslot/internal/failure"
	"github.com/ChiragAgg5k/mailslot/internal/mail"
)

// waitLateBackend has nothing on the first search and afterwards one message the server received before
// wait started, which Mail only now fetched.
type waitLateBackend struct {
	mail.Backend
	searches int
	received time.Time
}

func (backend *waitLateBackend) Fetch(context.Context, string) error {
	return nil
}

func (backend *waitLateBackend) Search(context.Context, mail.Query) ([]mail.Summary, error) {
	backend.searches++
	if backend.searches == 1 {
		return nil, nil
	}
	return []mail.Summary{{ID: 1, Subject: "Your verification code", Received: &backend.received}}, nil
}

func TestWaitAcceptsLateFetchWithinGrace(t *testing.T) {
	app := &application{backend: &waitLateBackend{received: time.Now().Add(-grace / 2)}}
	found, err := app.await(context.Background(), mail.Query{Subject: "verification"}, time.Second, 10*time.Millisecond)
	if err != nil || found.ID != 1 {
		t.Fatalf("await = %+v, %v; want message 1", found, err)
	}
}

func TestWaitIgnoresMessagesBeforeGrace(t *testing.T) {
	app := &application{backend: &waitLateBackend{received: time.Now().Add(-2 * grace)}}
	_, err := app.await(context.Background(), mail.Query{Subject: "verification"}, 50*time.Millisecond, 10*time.Millisecond)
	if failure.CodeOf(err) != failure.Busy {
		t.Fatalf("await error %v, want a timeout", err)
	}
}
