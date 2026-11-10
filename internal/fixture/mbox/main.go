// Command mbox writes one mailbox of the fixture data as an mbox file, so CI can import it into Mail as a local mailbox.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ChiragAgg5k/mailslot/internal/fixture"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "mbox:", err)
		os.Exit(1)
	}
}

func run() error {
	directory := flag.String("fixture", "testdata/fixture", "fixture directory to read the messages from")
	account := flag.String("account", "local", "account that owns the mailbox")
	mailbox := flag.String("mailbox", "Import/fixture", "mailbox whose messages to write")
	path := flag.String("output", "", "mbox file to write; parent directories are created")
	flag.StringVar(path, "o", "", "shorthand for -output")
	flag.Parse()
	if *path == "" {
		return errors.New("-output is required")
	}
	if err := os.MkdirAll(filepath.Dir(*path), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	file, err := os.Create(*path)
	if err != nil {
		return fmt.Errorf("create mbox: %w", err)
	}
	if err := fixture.WriteMbox(*directory, *account, *mailbox, file); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close mbox: %w", err)
	}
	return nil
}
