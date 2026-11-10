package fixture

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	netmail "net/mail"
	"net/textproto"
	"strings"
	"time"

	"github.com/ChiragAgg5k/mailslot/internal/mail"
)

const (
	lineEnd       = "\r\n"
	lineWidth     = 76
	charset       = "utf-8"
	plainText     = "text/plain; charset=utf-8"
	fallbackType  = "application/octet-stream"
	quotedPrinted = "quoted-printable"
	base64Name    = "base64"
)

// source renders the message as RFC 822 text, the way Mail's source property returns it.
func (message record) source() (string, error) {
	var output bytes.Buffer
	header := func(name, value string) {
		if value != "" {
			output.WriteString(name + ": " + value + lineEnd)
		}
	}
	header("From", encodeAddress(message.Sender))
	header("Reply-To", encodeAddress(message.ReplyTo))
	header("To", encodeAddresses(message.To))
	header("Cc", encodeAddresses(message.Cc))
	header("Subject", mime.QEncoding.Encode(charset, message.Subject))
	if when := message.date(); when != nil {
		header("Date", when.Format(time.RFC1123Z))
	}
	if message.MessageID != "" {
		header("Message-ID", "<"+message.MessageID+">")
	}
	if message.Headers != nil {
		header("In-Reply-To", message.Headers.InReplyTo)
		header("References", message.Headers.References)
	}
	header("MIME-Version", "1.0")

	if len(message.Attachments) == 0 {
		header("Content-Type", plainText)
		header("Content-Transfer-Encoding", quotedPrinted)
		output.WriteString(lineEnd)
		if err := writeText(&output, message.Content); err != nil {
			return "", err
		}
		return output.String(), nil
	}

	var body bytes.Buffer
	parts := multipart.NewWriter(&body)
	header("Content-Type", mime.FormatMediaType("multipart/mixed", map[string]string{"boundary": parts.Boundary()}))
	output.WriteString(lineEnd)

	text, err := parts.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {plainText},
		"Content-Transfer-Encoding": {quotedPrinted},
	})
	if err != nil {
		return "", fmt.Errorf("write message source: %w", err)
	}
	if err := writeText(text, message.Content); err != nil {
		return "", err
	}
	for _, attachment := range message.Attachments {
		kind := attachment.Type
		if kind == "" {
			kind = fallbackType
		}
		part, err := parts.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {mime.FormatMediaType(kind, map[string]string{"name": attachment.Name})},
			"Content-Disposition":       {mime.FormatMediaType("attachment", map[string]string{"filename": attachment.Name})},
			"Content-Transfer-Encoding": {base64Name},
		})
		if err != nil {
			return "", fmt.Errorf("write message source: %w", err)
		}
		encoded := base64.StdEncoding.EncodeToString([]byte(attachment.Data))
		for len(encoded) > lineWidth {
			fmt.Fprint(part, encoded[:lineWidth]+lineEnd)
			encoded = encoded[lineWidth:]
		}
		fmt.Fprint(part, encoded+lineEnd)
	}
	if err := parts.Close(); err != nil {
		return "", fmt.Errorf("write message source: %w", err)
	}
	output.Write(body.Bytes())
	return output.String(), nil
}

func (message record) date() *time.Time {
	if message.Sent != nil {
		return message.Sent
	}
	return message.Received
}

func writeText(output io.Writer, content string) error {
	writer := quotedprintable.NewWriter(output)
	if _, err := writer.Write([]byte(content)); err != nil {
		return fmt.Errorf("write message text: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("write message text: %w", err)
	}
	if !strings.HasSuffix(content, "\n") {
		_, err := output.Write([]byte(lineEnd))
		return err
	}
	return nil
}

func encodeAddress(raw string) string {
	if raw == "" {
		return ""
	}
	parsed, err := netmail.ParseAddress(raw)
	if err != nil {
		return mime.QEncoding.Encode(charset, raw)
	}
	return parsed.String()
}

func encodeAddresses(addresses []mail.Address) string {
	encoded := make([]string, 0, len(addresses))
	for _, address := range addresses {
		encoded = append(encoded, (&netmail.Address{Name: address.Name, Address: address.Address}).String())
	}
	return strings.Join(encoded, ", ")
}
