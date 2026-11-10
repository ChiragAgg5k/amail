package mail

import (
	"context"
	"fmt"
	netmail "net/mail"
	"strings"
)

const (
	replyPrefix = "Re: "
	quoteMark   = "> "
	quoteDate   = "2 Jan 2006 at 15:04"
)

// Draft is a message to compose. Unless Send is set it opens in Mail for review.
type Draft struct {
	Account     string   `json:"account"`
	To          []string `json:"to"`
	Cc          []string `json:"cc"`
	Bcc         []string `json:"bcc"`
	Subject     string   `json:"subject"`
	Body        string   `json:"body"`
	Attachments []string `json:"attachments"`
	Send        bool     `json:"send"`
}

// Delivery reports what Mail did with a draft.
type Delivery struct {
	Sent   bool   `json:"sent"`
	Sender string `json:"sender"`
}

// Compose hands a draft to Mail, which sends it or opens it for review.
func (client Client) Compose(ctx context.Context, draft Draft) (Delivery, error) {
	draft.To = present(draft.To)
	draft.Cc = present(draft.Cc)
	draft.Bcc = present(draft.Bcc)
	draft.Attachments = present(draft.Attachments)
	var delivery Delivery
	err := client.run(ctx, "compose", draft, &delivery)
	return delivery, err
}

// Reply builds an answer to the message with the original text quoted beneath the body.
//
// Mail cannot expose the quoted text of its own reply window to scripts, so the reply is
// a new message. It threads by subject and carries no In-Reply-To header.
func (message Message) Reply(body string, everyone bool, own []string) Draft {
	recipient := address(message.ReplyTo)
	if recipient == "" {
		recipient = address(message.Sender)
	}
	draft := Draft{
		To:      []string{recipient},
		Subject: message.Subject,
		Body:    strings.TrimRight(body, "\n") + "\n\n" + message.quoted(),
	}
	if !strings.HasPrefix(strings.ToLower(draft.Subject), strings.ToLower(strings.TrimSpace(replyPrefix))) {
		draft.Subject = replyPrefix + draft.Subject
	}
	if !everyone {
		return draft
	}

	seen := map[string]bool{strings.ToLower(recipient): true}
	for _, mine := range own {
		seen[strings.ToLower(mine)] = true
	}
	for _, other := range append(append([]Address{}, message.To...), message.Cc...) {
		key := strings.ToLower(other.Address)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		draft.Cc = append(draft.Cc, other.Address)
	}
	return draft
}

func (message Message) quoted() string {
	var quote strings.Builder
	when := ""
	if message.Received != nil {
		when = "On " + message.Received.Local().Format(quoteDate) + ", "
	}
	fmt.Fprintf(&quote, "%s%s wrote:\n", when, message.Sender)
	for _, line := range strings.Split(strings.TrimRight(message.Content, "\n"), "\n") {
		quote.WriteString(strings.TrimRight(quoteMark+line, " ") + "\n")
	}
	return quote.String()
}

func address(raw string) string {
	parsed, err := netmail.ParseAddress(raw)
	if err != nil {
		return strings.TrimSpace(raw)
	}
	return parsed.Address
}

func present(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
