// Package fixture is a mail backend that serves invented mail from JSON files, so every command can run without Mail.
package fixture

import (
	"context"
	"fmt"
	"mime"
	netmail "net/mail"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ChiragAgg5k/mailslot/internal/failure"
	"github.com/ChiragAgg5k/mailslot/internal/mail"
)

// Version is what Ping reports in place of Mail's version.
const Version = "fixture"

const (
	drafts        = "Drafts"
	signatureMark = "\n\n"
	sentDomain    = "fixture.example.com"
)

var sentCandidates = []string{"Sent", "Sent Messages"}

var _ mail.Backend = (*Backend)(nil)

// Backend answers every mail.Backend call from fixture data. Calls reload the data, so separate processes sharing a state file see each other's changes.
type Backend struct {
	state *state
	fault Fault
	mutex sync.Mutex
	clock func() time.Time
}

// Open serves the fixture data in directory. Changes are saved to the state file when one is given and kept in memory otherwise;
// the fixture directory itself is never written. The fault named by FaultVariable, if any, applies to every call.
func Open(directory, snapshot string) (*Backend, error) {
	fault, err := ParseFault(os.Getenv(FaultVariable))
	if err != nil {
		return nil, err
	}
	directory, err = filepath.Abs(directory)
	if err != nil {
		return nil, fmt.Errorf("locate fixture data: %w", err)
	}
	if snapshot != "" {
		snapshot, err = filepath.Abs(snapshot)
		if err != nil {
			return nil, fmt.Errorf("locate fixture state: %w", err)
		}
		if inside(directory, snapshot) {
			return nil, failure.New(failure.Usage, "the fixture state file must not live in the fixture directory", "point the state file somewhere else, such as a temporary directory")
		}
	}
	backend := &Backend{state: &state{directory: directory, path: snapshot}, fault: fault, clock: time.Now}
	if _, err := backend.state.load(); err != nil {
		return nil, err
	}
	return backend, nil
}

// inside reports whether the state file sits anywhere under the fixture directory, following symbolic links.
// The state file may not exist yet, so its nearest existing parent is resolved instead.
func inside(directory, snapshot string) bool {
	root, err := filepath.EvalSymlinks(directory)
	if err != nil {
		root = directory
	}
	parent, rest := filepath.Dir(snapshot), filepath.Base(snapshot)
	for {
		if resolved, err := filepath.EvalSymlinks(parent); err == nil {
			snapshot = filepath.Join(resolved, rest)
			break
		}
		above := filepath.Dir(parent)
		if above == parent {
			break
		}
		rest = filepath.Join(filepath.Base(parent), rest)
		parent = above
	}
	relative, err := filepath.Rel(root, snapshot)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func (backend *Backend) begin(ctx context.Context) (*data, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := backend.fault.err(); err != nil {
		return nil, err
	}
	return backend.state.load()
}

func (backend *Backend) now() *time.Time {
	now := backend.clock().UTC().Truncate(time.Second)
	return &now
}

// Ping reports the fixture version and the number of real accounts.
func (backend *Backend) Ping(ctx context.Context) (mail.Status, error) {
	backend.mutex.Lock()
	defer backend.mutex.Unlock()
	store, err := backend.begin(ctx)
	if err != nil {
		return mail.Status{}, err
	}
	return mail.Status{Version: Version, Accounts: len(store.Accounts)}, nil
}

// Accounts lists the real accounts, never local.
func (backend *Backend) Accounts(ctx context.Context) ([]mail.Account, error) {
	backend.mutex.Lock()
	defer backend.mutex.Unlock()
	store, err := backend.begin(ctx)
	if err != nil {
		return nil, err
	}
	return slices.Clone(store.Accounts), nil
}

// Mailboxes lists the mailboxes of an account with their message and unread counts.
func (backend *Backend) Mailboxes(ctx context.Context, account string) ([]mail.Mailbox, error) {
	backend.mutex.Lock()
	defer backend.mutex.Unlock()
	store, err := backend.begin(ctx)
	if err != nil {
		return nil, err
	}
	found, err := store.account(account)
	if err != nil {
		return nil, err
	}
	mailboxes := []mail.Mailbox{}
	for _, candidate := range store.folders(found.Name) {
		mailbox := mail.Mailbox{Name: candidate.Name}
		for _, index := range store.inside(found.Name, candidate.Name) {
			mailbox.Messages++
			if !store.Messages[index].Read {
				mailbox.Unread++
			}
		}
		mailboxes = append(mailboxes, mailbox)
	}
	return mailboxes, nil
}

// List returns the newest messages of a mailbox; a limit of 0 returns them all.
func (backend *Backend) List(ctx context.Context, location mail.Location, limit int, unread bool) ([]mail.Summary, error) {
	return backend.Search(ctx, mail.Query{Location: location, Unread: unread, Limit: limit})
}

// Search returns the messages of one mailbox matching every condition of the query, newest first.
func (backend *Backend) Search(ctx context.Context, query mail.Query) ([]mail.Summary, error) {
	backend.mutex.Lock()
	defer backend.mutex.Unlock()
	store, err := backend.begin(ctx)
	if err != nil {
		return nil, err
	}
	account, found, err := store.locate(query.Location)
	if err != nil {
		return nil, err
	}
	var matches []record
	for _, index := range store.inside(account.Name, found.Name) {
		if message := store.Messages[index]; matching(message, query) {
			matches = append(matches, message)
		}
	}
	slices.SortFunc(matches, func(first, second record) int {
		if first.newer(second) {
			return -1
		}
		if second.newer(first) {
			return 1
		}
		return 0
	})
	if query.Limit > 0 && len(matches) > query.Limit {
		matches = matches[:query.Limit]
	}
	summaries := make([]mail.Summary, 0, len(matches))
	for _, message := range matches {
		summaries = append(summaries, message.summary())
	}
	return summaries, nil
}

func matching(message record, query mail.Query) bool {
	if query.Text != "" && !contains(message.Subject, query.Text) && !contains(message.Sender, query.Text) &&
		(!query.Body || !contains(message.Content, query.Text)) {
		return false
	}
	if query.From != "" && !contains(message.Sender, query.From) {
		return false
	}
	if query.Subject != "" && !contains(message.Subject, query.Subject) {
		return false
	}
	if query.Unread && message.Read {
		return false
	}
	if query.Since != nil && (message.Received == nil || message.Received.Before(*query.Since)) {
		return false
	}
	return true
}

func contains(value, part string) bool {
	return strings.Contains(strings.ToLower(value), strings.ToLower(part))
}

// Resolve finds the targeted message and reports the mailbox it lives in.
func (backend *Backend) Resolve(ctx context.Context, target mail.Target) (mail.Summary, error) {
	message, err := backend.lookup(ctx, target)
	return message.summary(), err
}

// Read returns the full targeted message.
func (backend *Backend) Read(ctx context.Context, target mail.Target) (mail.Message, error) {
	message, err := backend.lookup(ctx, target)
	if err != nil {
		return mail.Message{}, err
	}
	return message.message(), nil
}

// Source returns the targeted message as RFC 822 text.
func (backend *Backend) Source(ctx context.Context, target mail.Target) (string, error) {
	message, err := backend.lookup(ctx, target)
	if err != nil {
		return "", err
	}
	return message.source()
}

// SaveAttachments writes the attachments of the targeted message into directory, renaming repeated names to name-2, name-3 and so on.
func (backend *Backend) SaveAttachments(ctx context.Context, target mail.Target, directory string) ([]mail.SavedAttachment, error) {
	message, err := backend.lookup(ctx, target)
	if err != nil {
		return nil, err
	}
	taken := map[string]bool{}
	saved := []mail.SavedAttachment{}
	for _, attachment := range message.Attachments {
		name := unused(attachment.Name, taken)
		taken[name] = true
		path := filepath.Join(directory, name)
		if err := os.WriteFile(path, []byte(attachment.Data), 0o644); err != nil {
			return saved, fmt.Errorf("save attachment %s: %w", name, err)
		}
		saved = append(saved, mail.SavedAttachment{Name: name, Path: path})
	}
	return saved, nil
}

func unused(name string, taken map[string]bool) string {
	if !taken[name] {
		return name
	}
	stem, extension := name, ""
	if dot := strings.LastIndex(name, "."); dot > 0 {
		stem, extension = name[:dot], name[dot:]
	}
	number := 2
	for taken[fmt.Sprintf("%s-%d%s", stem, number, extension)] {
		number++
	}
	return fmt.Sprintf("%s-%d%s", stem, number, extension)
}

func (backend *Backend) lookup(ctx context.Context, target mail.Target) (record, error) {
	backend.mutex.Lock()
	defer backend.mutex.Unlock()
	store, err := backend.begin(ctx)
	if err != nil {
		return record{}, err
	}
	index, err := store.find(target)
	if err != nil {
		return record{}, err
	}
	return store.Messages[index], nil
}

func (backend *Backend) change(ctx context.Context, target mail.Target, apply func(store *data, message *record) error) (mail.Summary, error) {
	backend.mutex.Lock()
	defer backend.mutex.Unlock()
	store, err := backend.begin(ctx)
	if err != nil {
		return mail.Summary{}, err
	}
	index, err := store.find(target)
	if err != nil {
		return mail.Summary{}, err
	}
	if err := apply(store, &store.Messages[index]); err != nil {
		return mail.Summary{}, err
	}
	if err := backend.state.save(store); err != nil {
		return mail.Summary{}, err
	}
	return store.Messages[index].summary(), nil
}

// Mark marks the targeted message read or unread.
func (backend *Backend) Mark(ctx context.Context, target mail.Target, read bool) (mail.Summary, error) {
	return backend.change(ctx, target, func(_ *data, message *record) error {
		message.Read = read
		return nil
	})
}

// Flag sets the flag colour of the targeted message, or clears it.
func (backend *Backend) Flag(ctx context.Context, target mail.Target, color mail.FlagColor) (mail.Summary, error) {
	return backend.change(ctx, target, func(_ *data, message *record) error {
		message.Flag = color
		return nil
	})
}

// Move moves the targeted message to another mailbox of its account, where it gets a new id as in Mail.
func (backend *Backend) Move(ctx context.Context, target mail.Target, destination string) (mail.Summary, error) {
	return backend.change(ctx, target, func(store *data, message *record) error {
		found, err := store.folder(message.Account, destination)
		if err != nil {
			return err
		}
		message.ID = store.nextID()
		message.Mailbox = found.Name
		return nil
	})
}

// Compose stores the draft in the account's Drafts mailbox, or in its sent mailbox when Send is set.
func (backend *Backend) Compose(ctx context.Context, draft mail.Draft) (mail.Delivery, error) {
	backend.mutex.Lock()
	defer backend.mutex.Unlock()
	store, err := backend.begin(ctx)
	if err != nil {
		return mail.Delivery{}, mail.Settle(err, draft.Send)
	}
	if strings.EqualFold(draft.Account, mail.Local) {
		return mail.Delivery{}, failure.New(failure.Refused, "local mailboxes cannot send mail", "")
	}
	account, err := store.account(draft.Account)
	if err != nil {
		return mail.Delivery{}, err
	}
	sender, err := from(account, draft.From)
	if err != nil {
		return mail.Delivery{}, err
	}
	content := draft.Body
	if draft.Signature != "" {
		chosen, err := store.signature(draft.Signature)
		if err != nil {
			return mail.Delivery{}, err
		}
		content = strings.TrimRight(content, "\n") + signatureMark + chosen.Content
	}
	attachments := make([]file, 0, len(draft.Attachments))
	for _, path := range draft.Attachments {
		contents, err := os.ReadFile(path)
		if err != nil {
			return mail.Delivery{}, failure.Wrap(failure.NotFound, fmt.Errorf("attach %s: %w", path, err), "check the attachment path")
		}
		attachments = append(attachments, file{Name: filepath.Base(path), Type: mime.TypeByExtension(filepath.Ext(path)), Data: string(contents)})
	}
	destination := store.destination(account.Name, draft.Send)
	id := store.nextID()
	now := backend.now()
	store.Messages = append(store.Messages, record{
		ID:          id,
		MessageID:   fmt.Sprintf("%d@%s", id, sentDomain),
		Account:     account.Name,
		Mailbox:     destination,
		Subject:     draft.Subject,
		Sender:      sender,
		To:          recipients(draft.To),
		Cc:          recipients(draft.Cc),
		Sent:        now,
		Received:    now,
		Read:        true,
		Flag:        mail.Clear,
		Content:     content,
		Attachments: attachments,
	})
	if err := backend.state.save(store); err != nil {
		return mail.Delivery{}, err
	}
	return mail.Delivery{Sent: draft.Send, Sender: sender}, nil
}

// from picks the sender as compose.js does: the account's first address by default, else the bare form of the requested one.
func from(account mail.Account, requested string) (string, error) {
	if requested == "" {
		if len(account.Addresses) == 0 {
			return "", nil
		}
		return account.Addresses[0], nil
	}
	address := requested
	if parsed, err := netmail.ParseAddress(requested); err == nil {
		address = parsed.Address
	}
	if !account.Owns(address) {
		return "", failure.New(failure.NotFound, fmt.Sprintf("account %q has no address %s", account.Name, requested), accountHint)
	}
	return address, nil
}

// destination names the mailbox a composed message lands in, adding it when the account has none.
func (store *data) destination(account string, send bool) string {
	candidates := []string{drafts}
	if send {
		candidates = sentCandidates
	}
	for _, candidate := range candidates {
		if found, err := store.folder(account, candidate); err == nil {
			return found.Name
		}
	}
	store.Mailboxes = append(store.Mailboxes, folder{Account: account, Name: candidates[0]})
	return candidates[0]
}

// Fetch delivers the account's incoming messages into its Inbox, unread and received now, once.
func (backend *Backend) Fetch(ctx context.Context, account string) error {
	backend.mutex.Lock()
	defer backend.mutex.Unlock()
	store, err := backend.begin(ctx)
	if err != nil {
		return err
	}
	found, err := store.account(account)
	if err != nil {
		return err
	}
	inbox, err := store.folder(found.Name, mail.Inbox)
	if err != nil {
		return err
	}
	remaining := []record{}
	delivered := false
	now := backend.now()
	for _, message := range store.Incoming {
		if !strings.EqualFold(message.Account, found.Name) {
			remaining = append(remaining, message)
			continue
		}
		if store.taken(message.ID) {
			message.ID = store.nextID()
		}
		message.Account = found.Name
		message.Mailbox = inbox.Name
		message.Received = now
		message.Read = false
		store.Messages = append(store.Messages, message)
		delivered = true
	}
	if !delivered {
		return nil
	}
	store.Incoming = remaining
	return backend.state.save(store)
}
