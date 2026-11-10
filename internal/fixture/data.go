package fixture

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ChiragAgg5k/mailslot/internal/failure"
	"github.com/ChiragAgg5k/mailslot/internal/mail"
)

const (
	accountHint = "run mailslot accounts to see the account names"
	mailboxHint = "run mailslot mailboxes to see the mailbox names"
)

// data is the whole invented mail store: what the fixture directory holds and what a state snapshot saves.
type data struct {
	Accounts   []mail.Account `json:"accounts"`
	Signatures []signature    `json:"signatures"`
	Mailboxes  []folder       `json:"mailboxes"`
	Messages   []record       `json:"messages"`
	Incoming   []record       `json:"incoming"`
}

func (store *data) account(name string) (mail.Account, error) {
	if strings.EqualFold(name, mail.Local) {
		return mail.LocalAccount(), nil
	}
	for _, account := range store.Accounts {
		if strings.EqualFold(account.Name, name) {
			return account, nil
		}
	}
	return mail.Account{}, failure.New(failure.NotFound, fmt.Sprintf("no account named %q", name), accountHint)
}

func (store *data) folders(account string) []folder {
	var owned []folder
	for _, candidate := range store.Mailboxes {
		if strings.EqualFold(candidate.Account, account) {
			owned = append(owned, candidate)
		}
	}
	return owned
}

// folder finds a mailbox by name, ignoring case. As in findMailbox of prelude.js, a nested mailbox of any account
// also answers to its leaf name when no full name matches.
func (store *data) folder(account, name string) (folder, error) {
	if name == "" {
		name = mail.Inbox
	}
	owned := store.folders(account)
	for _, candidate := range owned {
		if strings.EqualFold(candidate.Name, name) {
			return candidate, nil
		}
	}
	for _, candidate := range owned {
		if strings.EqualFold(candidate.leaf(), name) {
			return candidate, nil
		}
	}
	return folder{}, failure.New(failure.NotFound, fmt.Sprintf("no mailbox named %q in account %q", name, account), mailboxHint)
}

func (store *data) locate(location mail.Location) (mail.Account, folder, error) {
	account, err := store.account(location.Account)
	if err != nil {
		return mail.Account{}, folder{}, err
	}
	found, err := store.folder(account.Name, location.Mailbox)
	return account, found, err
}

func (store *data) inside(account, mailbox string) []int {
	var indexes []int
	for index, message := range store.Messages {
		if strings.EqualFold(message.Account, account) && strings.EqualFold(message.Mailbox, mailbox) {
			indexes = append(indexes, index)
		}
	}
	return indexes
}

// find returns the index of the targeted message. An empty mailbox searches every mailbox of the account, Inbox first.
func (store *data) find(target mail.Target) (int, error) {
	account, err := store.account(target.Account)
	if err != nil {
		return 0, err
	}
	var scope []folder
	if target.Mailbox != "" {
		found, err := store.folder(account.Name, target.Mailbox)
		if err != nil {
			return 0, err
		}
		scope = []folder{found}
	} else {
		owned := store.folders(account.Name)
		for _, candidate := range owned {
			if strings.EqualFold(candidate.Name, mail.Inbox) {
				scope = append(scope, candidate)
			}
		}
		for _, candidate := range owned {
			if !strings.EqualFold(candidate.Name, mail.Inbox) {
				scope = append(scope, candidate)
			}
		}
	}
	for _, candidate := range scope {
		for _, index := range store.inside(account.Name, candidate.Name) {
			message := store.Messages[index]
			if target.ID > 0 && message.ID == target.ID {
				return index, nil
			}
			if target.ID <= 0 && target.MessageID != "" && message.MessageID == target.MessageID {
				return index, nil
			}
		}
	}
	return 0, mail.MessageNotFound(target, "no message "+describe(target))
}

func (store *data) nextID() int {
	highest := 0
	for _, group := range [][]record{store.Messages, store.Incoming} {
		for _, message := range group {
			highest = max(highest, message.ID)
		}
	}
	return highest + 1
}

func (store *data) taken(id int) bool {
	for _, message := range store.Messages {
		if message.ID == id {
			return true
		}
	}
	return false
}

func (store *data) signature(name string) (signature, error) {
	for _, candidate := range store.Signatures {
		if strings.EqualFold(candidate.Name, name) {
			return candidate, nil
		}
	}
	return signature{}, failure.New(failure.NotFound, fmt.Sprintf("no signature named %q", name), "check the signature names under Mail → Settings → Signatures")
}

func describe(target mail.Target) string {
	reference := "with Message-ID <" + target.MessageID + ">"
	if target.ID > 0 {
		reference = strconv.Itoa(target.ID)
	}
	where := "any mailbox"
	if target.Mailbox != "" {
		where = fmt.Sprintf("mailbox %q", target.Mailbox)
	}
	return fmt.Sprintf("%s in %s of account %q", reference, where, target.Account)
}
