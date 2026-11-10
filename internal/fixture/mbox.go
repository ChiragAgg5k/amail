package fixture

import (
	"bufio"
	"fmt"
	"io"
	netmail "net/mail"
	"strings"
	"time"
)

const (
	asctime       = "Mon Jan _2 15:04:05 2006"
	unknownSender = "MAILER-DAEMON"
	separator     = "From "
	quote         = ">"
)

// WriteMbox writes every message of one account's mailbox in the fixture directory as an mboxrd file, the format Mail's import reads.
func WriteMbox(directory, account, mailbox string, output io.Writer) error {
	store, err := (&state{directory: directory}).read()
	if err != nil {
		return err
	}
	indexes := store.inside(account, mailbox)
	if len(indexes) == 0 {
		return fmt.Errorf("write mbox: no messages in mailbox %q of account %q", mailbox, account)
	}
	writer := bufio.NewWriter(output)
	for _, index := range indexes {
		if err := writeEntry(writer, store.Messages[index]); err != nil {
			return err
		}
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("write mbox: %w", err)
	}
	return nil
}

func writeEntry(writer *bufio.Writer, message record) error {
	source, err := message.source()
	if err != nil {
		return fmt.Errorf("write mbox message %d: %w", message.ID, err)
	}
	when := time.Unix(0, 0)
	if date := message.date(); date != nil {
		when = *date
	}
	writer.WriteString(separator + envelopeSender(message.Sender) + " " + when.UTC().Format(asctime) + "\n")
	source = strings.TrimRight(strings.ReplaceAll(source, lineEnd, "\n"), "\n")
	for line := range strings.SplitSeq(source, "\n") {
		if strings.HasPrefix(strings.TrimLeft(line, quote), separator) {
			writer.WriteString(quote)
		}
		writer.WriteString(line + "\n")
	}
	_, err = writer.WriteString("\n")
	if err != nil {
		return fmt.Errorf("write mbox message %d: %w", message.ID, err)
	}
	return nil
}

func envelopeSender(sender string) string {
	parsed, err := netmail.ParseAddress(sender)
	if err != nil || parsed.Address == "" || strings.ContainsAny(parsed.Address, " \t") {
		return unknownSender
	}
	return parsed.Address
}
