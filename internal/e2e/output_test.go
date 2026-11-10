//go:build e2e

package e2e_test

import "time"

// The types below mirror mailslot's JSON output, so a renamed or retyped field fails the suite.

type account struct {
	Name      string   `json:"name"`
	Addresses []string `json:"addresses"`
	Kind      string   `json:"kind"`
	Enabled   bool     `json:"enabled"`
}

type mailbox struct {
	Name     string `json:"name"`
	Unread   int    `json:"unread"`
	Messages int    `json:"messages"`
}

type summary struct {
	ID        int        `json:"id"`
	MessageID string     `json:"messageId"`
	Mailbox   string     `json:"mailbox"`
	Subject   string     `json:"subject"`
	Sender    string     `json:"sender"`
	Received  *time.Time `json:"received"`
	Read      bool       `json:"read"`
	Flagged   bool       `json:"flagged"`
}

type attachment struct {
	Name string `json:"name"`
	Size int    `json:"size"`
	Type string `json:"type"`
}

type message struct {
	summary
	ReplyTo     string       `json:"replyTo"`
	Content     string       `json:"content"`
	Attachments []attachment `json:"attachments"`
}

type saved struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type source struct {
	ID        int    `json:"id"`
	MessageID string `json:"messageId"`
	Mailbox   string `json:"mailbox"`
	Source    string `json:"source"`
	Path      string `json:"path"`
}

type change struct {
	ID        int    `json:"id"`
	MessageID string `json:"messageId"`
	Mailbox   string `json:"mailbox"`
	Change    string `json:"change"`
	NewID     *int   `json:"newId"`
	DryRun    bool   `json:"dryRun"`
}

type check struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

type diagnosis struct {
	Checks []check `json:"checks"`
}

type delivery struct {
	Sent   bool   `json:"sent"`
	Sender string `json:"sender"`
}

type arrival struct {
	Message summary `json:"message"`
	Code    string  `json:"code"`
}

type merged struct {
	Row     int      `json:"row"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
}
