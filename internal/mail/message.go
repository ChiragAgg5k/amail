package mail

import (
	"context"
	"time"
)

// Summary is one row of a message listing.
type Summary struct {
	ID       int        `json:"id"`
	Subject  string     `json:"subject"`
	Sender   string     `json:"sender"`
	Received *time.Time `json:"received"`
	Read     bool       `json:"read"`
	Flagged  bool       `json:"flagged"`
}

// Address is a named mail recipient.
type Address struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

// Message is a full message, including its text.
type Message struct {
	ID          int          `json:"id"`
	MessageID   string       `json:"messageId"`
	Subject     string       `json:"subject"`
	Sender      string       `json:"sender"`
	ReplyTo     string       `json:"replyTo"`
	To          []Address    `json:"to"`
	Cc          []Address    `json:"cc"`
	Sent        *time.Time   `json:"sent"`
	Received    *time.Time   `json:"received"`
	Read        bool         `json:"read"`
	Flagged     bool         `json:"flagged"`
	Content     string       `json:"content"`
	Attachments []Attachment `json:"attachments"`
}

// Query narrows a search. Empty fields are ignored.
type Query struct {
	Location
	Text    string     `json:"text"`
	From    string     `json:"from"`
	Subject string     `json:"subject"`
	Unread  bool       `json:"unread"`
	Since   *time.Time `json:"since"`
	Limit   int        `json:"limit"`
}

type listing struct {
	Location
	Unread bool `json:"unread"`
	Limit  int  `json:"limit"`
}

type reference struct {
	Location
	ID int `json:"id"`
}

// List returns the newest messages of a mailbox.
func (client Client) List(ctx context.Context, location Location, limit int, unread bool) ([]Summary, error) {
	var summaries []Summary
	err := client.run(ctx, "list", listing{Location: location, Unread: unread, Limit: limit}, &summaries)
	return summaries, err
}

// Search returns the messages of a mailbox that match the query, newest first.
func (client Client) Search(ctx context.Context, query Query) ([]Summary, error) {
	var summaries []Summary
	err := client.run(ctx, "search", query, &summaries)
	return summaries, err
}

// Read returns one message by its Mail identifier.
func (client Client) Read(ctx context.Context, location Location, id int) (Message, error) {
	var message Message
	err := client.run(ctx, "read", reference{Location: location, ID: id}, &message)
	return message, err
}
