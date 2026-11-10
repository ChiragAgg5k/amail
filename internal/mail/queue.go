package mail

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ChiragAgg5k/mailslot/internal/lock"
)

const (
	cacheDirectory  = "mailslot"
	lockFile        = "mail.lock"
	fixtureLockFile = "fixture.lock"
)

// Queue lets one call at a time through to its backend, across processes, because Mail answers one request at a time.
type Queue struct {
	Backend Backend
	Path    string
	// Timeout bounds the wait for the lock; zero means DefaultTimeout.
	Timeout time.Duration
}

var _ Backend = Queue{}

// LockPath is where mailslot commands queue for Mail.
func LockPath() (string, error) {
	return cachePath(lockFile)
}

// FixtureLockPath is where commands running against a fixture queue, apart from the ones talking to
// Mail, so a test or recording never waits for a real command.
func FixtureLockPath() (string, error) {
	return cachePath(fixtureLockFile)
}

func cachePath(name string) (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("find cache directory: %w", err)
	}
	return filepath.Join(base, cacheDirectory, name), nil
}

func queued[Result any](queue Queue, ctx context.Context, call func() (Result, error)) (result Result, err error) {
	timeout := queue.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}
	held, err := lock.Acquire(ctx, queue.Path, timeout)
	if err != nil {
		return result, err
	}
	defer func() {
		if released := held.Release(); released != nil {
			err = errors.Join(err, fmt.Errorf("release lock: %w", released))
		}
	}()
	return call()
}

func (queue Queue) Ping(ctx context.Context) (Status, error) {
	return queued(queue, ctx, func() (Status, error) { return queue.Backend.Ping(ctx) })
}

func (queue Queue) Accounts(ctx context.Context) ([]Account, error) {
	return queued(queue, ctx, func() ([]Account, error) { return queue.Backend.Accounts(ctx) })
}

func (queue Queue) Mailboxes(ctx context.Context, account string) ([]Mailbox, error) {
	return queued(queue, ctx, func() ([]Mailbox, error) { return queue.Backend.Mailboxes(ctx, account) })
}

func (queue Queue) List(ctx context.Context, location Location, limit int, unread bool) ([]Summary, error) {
	return queued(queue, ctx, func() ([]Summary, error) { return queue.Backend.List(ctx, location, limit, unread) })
}

func (queue Queue) Search(ctx context.Context, query Query) ([]Summary, error) {
	return queued(queue, ctx, func() ([]Summary, error) { return queue.Backend.Search(ctx, query) })
}

func (queue Queue) Resolve(ctx context.Context, target Target) (Summary, error) {
	return queued(queue, ctx, func() (Summary, error) { return queue.Backend.Resolve(ctx, target) })
}

func (queue Queue) Read(ctx context.Context, target Target) (Message, error) {
	return queued(queue, ctx, func() (Message, error) { return queue.Backend.Read(ctx, target) })
}

func (queue Queue) Source(ctx context.Context, target Target) (string, error) {
	return queued(queue, ctx, func() (string, error) { return queue.Backend.Source(ctx, target) })
}

func (queue Queue) SaveAttachments(ctx context.Context, target Target, directory string) ([]SavedAttachment, error) {
	return queued(queue, ctx, func() ([]SavedAttachment, error) { return queue.Backend.SaveAttachments(ctx, target, directory) })
}

func (queue Queue) Mark(ctx context.Context, target Target, read bool) (Summary, error) {
	return queued(queue, ctx, func() (Summary, error) { return queue.Backend.Mark(ctx, target, read) })
}

func (queue Queue) Flag(ctx context.Context, target Target, color FlagColor) (Summary, error) {
	return queued(queue, ctx, func() (Summary, error) { return queue.Backend.Flag(ctx, target, color) })
}

func (queue Queue) Move(ctx context.Context, target Target, destination string) (Summary, error) {
	return queued(queue, ctx, func() (Summary, error) { return queue.Backend.Move(ctx, target, destination) })
}

func (queue Queue) Compose(ctx context.Context, draft Draft) (Delivery, error) {
	return queued(queue, ctx, func() (Delivery, error) { return queue.Backend.Compose(ctx, draft) })
}

func (queue Queue) Fetch(ctx context.Context, account string) error {
	_, err := queued(queue, ctx, func() (struct{}, error) { return struct{}{}, queue.Backend.Fetch(ctx, account) })
	return err
}
