// Package fixture is a placeholder for the fixture backend, which replaces this file when the branches merge.
package fixture

import (
	"context"
	"errors"

	"github.com/ChiragAgg5k/mailslot/internal/mail"
)

var errUnbuilt = errors.New("fixture backend not built")

// Backend stands in for the fixture backend.
type Backend struct{}

// Open always fails until the real fixture backend is merged.
func Open(directory, state string) (*Backend, error) {
	return nil, errUnbuilt
}

func (backend *Backend) Ping(context.Context) (mail.Status, error) {
	return mail.Status{}, errUnbuilt
}

func (backend *Backend) Accounts(context.Context) ([]mail.Account, error) {
	return nil, errUnbuilt
}

func (backend *Backend) Mailboxes(context.Context, string) ([]mail.Mailbox, error) {
	return nil, errUnbuilt
}

func (backend *Backend) List(context.Context, mail.Location, int, bool) ([]mail.Summary, error) {
	return nil, errUnbuilt
}

func (backend *Backend) Search(context.Context, mail.Query) ([]mail.Summary, error) {
	return nil, errUnbuilt
}

func (backend *Backend) Resolve(context.Context, mail.Target) (mail.Summary, error) {
	return mail.Summary{}, errUnbuilt
}

func (backend *Backend) Read(context.Context, mail.Target) (mail.Message, error) {
	return mail.Message{}, errUnbuilt
}

func (backend *Backend) Source(context.Context, mail.Target) (string, error) {
	return "", errUnbuilt
}

func (backend *Backend) SaveAttachments(context.Context, mail.Target, string) ([]mail.SavedAttachment, error) {
	return nil, errUnbuilt
}

func (backend *Backend) Mark(context.Context, mail.Target, bool) (mail.Summary, error) {
	return mail.Summary{}, errUnbuilt
}

func (backend *Backend) Flag(context.Context, mail.Target, mail.FlagColor) (mail.Summary, error) {
	return mail.Summary{}, errUnbuilt
}

func (backend *Backend) Move(context.Context, mail.Target, string) (mail.Summary, error) {
	return mail.Summary{}, errUnbuilt
}

func (backend *Backend) Compose(context.Context, mail.Draft) (mail.Delivery, error) {
	return mail.Delivery{}, errUnbuilt
}

func (backend *Backend) Fetch(context.Context, string) error {
	return errUnbuilt
}
