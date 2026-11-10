// Package lock queues mailslot commands so that only one talks to Mail at a time.
package lock

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ChiragAgg5k/mailslot/internal/failure"
)

const interval = 100 * time.Millisecond

// Lock is an exclusive advisory lock on a file.
type Lock struct {
	file *os.File
}

// Acquire waits up to timeout for the lock at path, creating the file and its directory when needed.
func Acquire(ctx context.Context, path string, timeout time.Duration) (*Lock, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create lock directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return &Lock{file: file}, nil
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) && !errors.Is(err, syscall.EINTR) {
			file.Close()
			return nil, fmt.Errorf("lock %s: %w", path, err)
		}
		select {
		case <-ctx.Done():
			file.Close()
			return nil, ctx.Err()
		case <-deadline.C:
			file.Close()
			return nil, failure.New(failure.Busy, "another mailslot command is still talking to Mail", "wait for it to finish and try again")
		case <-ticker.C:
		}
	}
}

// Release frees the lock for the next command.
func (lock *Lock) Release() error {
	unlock := syscall.Flock(int(lock.file.Fd()), syscall.LOCK_UN)
	return errors.Join(unlock, lock.file.Close())
}
