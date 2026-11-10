package mail

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ChiragAgg5k/mailslot/internal/failure"
)

const (
	messageIDMark  = "@"
	openingBracket = "<"
	closingBracket = ">"
)

// Reference is a message reference: Mail's numeric id, or the Message-ID header.
type Reference struct {
	ID int `json:"id"`
	// MessageID is stored without angle brackets, as Mail's messageId() reports it.
	MessageID string `json:"messageId"`
}

// Target names one message in one place.
type Target struct {
	Location
	Reference
}

// ParseReference reads a message reference: a Message-ID (with or without angle brackets) or a positive numeric id.
func ParseReference(value string) (Reference, error) {
	trimmed := strings.TrimSpace(value)
	if strings.Contains(trimmed, messageIDMark) || (strings.HasPrefix(trimmed, openingBracket) && strings.HasSuffix(trimmed, closingBracket)) {
		identifier := strings.TrimSuffix(strings.TrimPrefix(trimmed, openingBracket), closingBracket)
		if identifier == "" {
			return Reference{}, invalidReference(value)
		}
		return Reference{MessageID: identifier}, nil
	}
	id, err := strconv.Atoi(trimmed)
	if err != nil || id <= 0 {
		return Reference{}, invalidReference(value)
	}
	return Reference{ID: id}, nil
}

func invalidReference(value string) *failure.Error {
	return failure.New(failure.Usage, fmt.Sprintf("%q is not a message reference", value), "pass a numeric id or a Message-ID from `mailslot list`")
}

// MessageNotFound reports a target that no longer names a message.
func MessageNotFound(target Target, detail string) *failure.Error {
	if target.MessageID == "" {
		return failure.New(failure.NotFound, detail, "a numeric id only holds inside the mailbox it was listed from and changes when the message moves; retry with its Message-ID (the messageId field)")
	}
	return failure.New(failure.NotFound, detail, "check --account and --mailbox; the message may have been deleted")
}
