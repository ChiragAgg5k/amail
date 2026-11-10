package fixture_test

import (
	"bytes"
	"io"
	netmail "net/mail"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/ChiragAgg5k/mailslot/internal/fixture"
)

var mboxSeparator = regexp.MustCompile(`(?m)^From \S+ \w{3} \w{3} [ \d]\d \d{2}:\d{2}:\d{2} \d{4}\n`)

func TestMboxImportFixture(t *testing.T) {
	var output bytes.Buffer
	if err := fixture.WriteMbox(filepath.Join("..", "..", "testdata", "fixture"), "local", "Import/fixture", &output); err != nil {
		t.Fatalf("write mbox: %v", err)
	}
	mbox := output.String()
	if strings.Contains(mbox, "\r") {
		t.Error("mbox contains carriage returns")
	}
	if !strings.HasPrefix(mbox, "From ") {
		t.Fatalf("mbox does not start with a separator: %q", mbox[:min(len(mbox), 40)])
	}
	entries := mboxSeparator.Split(mbox, -1)[1:]
	if len(entries) != 4 {
		t.Fatalf("got %d messages, want 4", len(entries))
	}
	var identifiers []string
	attachments := 0
	for index, entry := range entries {
		message, err := netmail.ReadMessage(strings.NewReader(entry))
		if err != nil {
			t.Fatalf("parse message %d: %v", index, err)
		}
		identifiers = append(identifiers, message.Header.Get("Message-Id"))
		if strings.HasPrefix(message.Header.Get("Content-Type"), "multipart/mixed") {
			attachments++
		}
		if _, err := io.ReadAll(message.Body); err != nil {
			t.Fatalf("read message %d body: %v", index, err)
		}
	}
	want := []string{
		"<import-1024@mail.example.com>",
		"<import-1025@mail.example.com>",
		"<import-1026@mail.example.com>",
		"<import-1027@mail.example.org>",
	}
	if !slices.Equal(identifiers, want) {
		t.Errorf("Message-IDs = %v, want %v", identifiers, want)
	}
	if attachments != 1 {
		t.Errorf("got %d messages with attachments, want 1", attachments)
	}
	if !strings.Contains(mbox, "mailslot-e2e-marker") {
		t.Error("mbox lacks the mailslot-e2e-marker subject")
	}
}

func TestMboxQuotesFromLines(t *testing.T) {
	var output bytes.Buffer
	if err := fixture.WriteMbox(filepath.Join("..", "..", "testdata", "fixture"), "local", "Import/fixture", &output); err != nil {
		t.Fatalf("write mbox: %v", err)
	}
	for line := range strings.SplitSeq(output.String(), "\n") {
		if strings.HasPrefix(line, "From ") && !mboxSeparator.MatchString(line+"\n") {
			t.Errorf("unquoted From line inside a message: %q", line)
		}
	}
}

func TestMboxMissingMailbox(t *testing.T) {
	err := fixture.WriteMbox(filepath.Join("..", "..", "testdata", "fixture"), "local", "Nowhere", io.Discard)
	if err == nil {
		t.Fatal("expected an error for a mailbox without messages")
	}
}
