package cli

import (
	"encoding/json"
	"fmt"
	"io"
	netmail "net/mail"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/ChiragAgg5k/mailslot/internal/failure"
	"github.com/ChiragAgg5k/mailslot/internal/mail"
)

const (
	timeLayout = "2006-01-02 15:04"
	unreadMark = "*"
	flagMark   = "!"
)

func (app *application) print(output io.Writer, value any, table func(io.Writer)) error {
	if !app.json {
		table(output)
		return nil
	}
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}

func columns(output io.Writer, header string, rows func(io.Writer)) {
	writer := tabwriter.NewWriter(output, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, header)
	rows(writer)
	writer.Flush()
}

func summaries(output io.Writer, rows []mail.Summary) {
	if len(rows) == 0 {
		fmt.Fprintln(output, "No messages.")
		return
	}
	spread := false
	for _, row := range rows {
		if row.Mailbox != rows[0].Mailbox {
			spread = true
			break
		}
	}
	header := "ID\t\tRECEIVED\tFROM\tSUBJECT"
	if spread {
		header = "ID\t\tRECEIVED\tMAILBOX\tFROM\tSUBJECT"
	}
	columns(output, header, func(writer io.Writer) {
		for _, row := range rows {
			marks := ""
			if !row.Read {
				marks += unreadMark
			}
			if row.Flagged {
				marks += flagMark
			}
			if spread {
				fmt.Fprintf(writer, "%d\t%s\t%s\t%s\t%s\t%s\n", row.ID, marks, local(row.Received), row.Mailbox, display(row.Sender), row.Subject)
				continue
			}
			fmt.Fprintf(writer, "%d\t%s\t%s\t%s\t%s\n", row.ID, marks, local(row.Received), display(row.Sender), row.Subject)
		}
	})
}

// problem is the JSON form of an error.
type problem struct {
	Error string       `json:"error"`
	Code  failure.Code `json:"code"`
	Hint  string       `json:"hint"`
}

// report writes an error to standard error, as JSON when asked for.
func report(output io.Writer, err error, asJSON bool) {
	hint := failure.HintOf(err)
	if asJSON {
		encoder := json.NewEncoder(output)
		encoder.SetEscapeHTML(false)
		if encoder.Encode(problem{Error: err.Error(), Code: failure.CodeOf(err), Hint: hint}) == nil {
			return
		}
	}
	fmt.Fprintln(output, "mailslot:", err)
	if hint != "" {
		fmt.Fprintln(output, "hint:", hint)
	}
}

func local(moment *time.Time) string {
	if moment == nil {
		return ""
	}
	return moment.Local().Format(timeLayout)
}

func display(sender string) string {
	parsed, err := netmail.ParseAddress(sender)
	if err != nil {
		return sender
	}
	if parsed.Name != "" {
		return parsed.Name
	}
	return parsed.Address
}

func recipients(addresses []mail.Address) string {
	parts := make([]string, 0, len(addresses))
	for _, address := range addresses {
		if address.Name == "" || address.Name == address.Address {
			parts = append(parts, address.Address)
			continue
		}
		parts = append(parts, fmt.Sprintf("%s <%s>", address.Name, address.Address))
	}
	return strings.Join(parts, ", ")
}
