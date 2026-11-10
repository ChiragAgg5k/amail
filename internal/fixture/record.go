package fixture

import (
	"encoding/json"
	netmail "net/mail"
	"time"

	"github.com/ChiragAgg5k/mailslot/internal/mail"
)

// record is one message as the fixture data stores it.
type record struct {
	ID          int            `json:"id"`
	MessageID   string         `json:"messageId"`
	Account     string         `json:"account"`
	Mailbox     string         `json:"mailbox"`
	Subject     string         `json:"subject"`
	Sender      string         `json:"sender"`
	ReplyTo     string         `json:"replyTo"`
	To          []mail.Address `json:"to"`
	Cc          []mail.Address `json:"cc"`
	Sent        *time.Time     `json:"sent"`
	Received    *time.Time     `json:"received"`
	Read        bool           `json:"read"`
	Flag        mail.FlagColor `json:"flag"`
	Content     string         `json:"content"`
	Headers     *headers       `json:"headers,omitempty"`
	Attachments []file         `json:"attachments,omitempty"`
}

// UnmarshalJSON treats a missing flag as no flag rather than red, the zero colour.
func (message *record) UnmarshalJSON(data []byte) error {
	type plain record
	decoded := plain{Flag: mail.Clear}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*message = record(decoded)
	return nil
}

func (message record) summary() mail.Summary {
	return mail.Summary{
		ID:        message.ID,
		MessageID: message.MessageID,
		Mailbox:   message.Mailbox,
		Subject:   message.Subject,
		Sender:    message.Sender,
		Received:  message.Received,
		Read:      message.Read,
		Flagged:   message.Flag != mail.Clear,
	}
}

func (message record) message() mail.Message {
	attachments := make([]mail.Attachment, 0, len(message.Attachments))
	for _, attachment := range message.Attachments {
		attachments = append(attachments, mail.Attachment{Name: attachment.Name, Size: len(attachment.Data), Type: attachment.Type})
	}
	return mail.Message{
		ID:          message.ID,
		MessageID:   message.MessageID,
		Mailbox:     message.Mailbox,
		Subject:     message.Subject,
		Sender:      message.Sender,
		ReplyTo:     message.ReplyTo,
		To:          listed(message.To),
		Cc:          listed(message.Cc),
		Sent:        message.Sent,
		Received:    message.Received,
		Read:        message.Read,
		Flagged:     message.Flag != mail.Clear,
		Content:     message.Content,
		Attachments: attachments,
	}
}

func (message record) newer(other record) bool {
	if message.Received == nil || other.Received == nil {
		if message.Received != other.Received {
			return other.Received == nil
		}
		return message.ID > other.ID
	}
	if message.Received.Equal(*other.Received) {
		return message.ID > other.ID
	}
	return message.Received.After(*other.Received)
}

func listed(addresses []mail.Address) []mail.Address {
	if addresses == nil {
		return []mail.Address{}
	}
	return addresses
}

func recipients(values []string) []mail.Address {
	addresses := make([]mail.Address, 0, len(values))
	for _, value := range values {
		parsed, err := netmail.ParseAddress(value)
		if err != nil {
			addresses = append(addresses, mail.Address{Address: value})
			continue
		}
		addresses = append(addresses, mail.Address{Name: parsed.Name, Address: parsed.Address})
	}
	return addresses
}
