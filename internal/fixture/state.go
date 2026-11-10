package fixture

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const (
	accountsFile   = "accounts.json"
	signaturesFile = "signatures.json"
	mailboxesFile  = "mailboxes.json"
	messagesFile   = "messages.json"
	incomingFile   = "incoming.json"
	temporaryName  = ".mailslot-state-*"
)

// state loads the fixture data and keeps what commands change: in a snapshot file when it has a path, otherwise in memory.
type state struct {
	directory string
	path      string
	memory    *data
}

func (keeper *state) load() (*data, error) {
	if keeper.path != "" {
		encoded, err := os.ReadFile(keeper.path)
		switch {
		case err == nil:
			var snapshot data
			if err := json.Unmarshal(encoded, &snapshot); err != nil {
				return nil, fmt.Errorf("decode fixture state %s: %w", keeper.path, err)
			}
			return &snapshot, nil
		case !errors.Is(err, fs.ErrNotExist):
			return nil, fmt.Errorf("read fixture state: %w", err)
		}
	} else if keeper.memory != nil {
		return keeper.memory, nil
	}
	return keeper.read()
}

func (keeper *state) read() (*data, error) {
	store := &data{}
	files := []struct {
		name     string
		target   any
		required bool
	}{
		{accountsFile, &store.Accounts, true},
		{signaturesFile, &store.Signatures, false},
		{mailboxesFile, &store.Mailboxes, true},
		{messagesFile, &store.Messages, true},
		{incomingFile, &store.Incoming, false},
	}
	for _, entry := range files {
		path := filepath.Join(keeper.directory, entry.name)
		encoded, err := os.ReadFile(path)
		if err != nil {
			if !entry.required && errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("read fixture data: %w", err)
		}
		if err := json.Unmarshal(encoded, entry.target); err != nil {
			return nil, fmt.Errorf("decode fixture data %s: %w", path, err)
		}
	}
	for index := range store.Accounts {
		if store.Accounts[index].Addresses == nil {
			store.Accounts[index].Addresses = []string{}
		}
	}
	return store, nil
}

func (keeper *state) save(store *data) error {
	if keeper.path == "" {
		keeper.memory = store
		return nil
	}
	encoded, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("encode fixture state: %w", err)
	}
	directory := filepath.Dir(keeper.path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create fixture state directory: %w", err)
	}
	temporary, err := os.CreateTemp(directory, temporaryName)
	if err != nil {
		return fmt.Errorf("create fixture state: %w", err)
	}
	defer os.Remove(temporary.Name())
	if _, err := temporary.Write(encoded); err != nil {
		temporary.Close()
		return fmt.Errorf("write fixture state: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("write fixture state: %w", err)
	}
	if err := os.Rename(temporary.Name(), keeper.path); err != nil {
		return fmt.Errorf("replace fixture state: %w", err)
	}
	return nil
}
