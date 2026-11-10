package cli

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ChiragAgg5k/mailslot/internal/failure"
	"github.com/ChiragAgg5k/mailslot/internal/mail"
	"github.com/ChiragAgg5k/mailslot/internal/merge"
)

// mailing holds the templated fields of a merge, one draft per CSV row.
type mailing struct {
	to      []string
	cc      []string
	subject string
	dryRun  bool
}

// merged is one row of a merge's report.
type merged struct {
	Row     int      `json:"row"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	// Body is reported on a dry run only.
	Body string `json:"body,omitempty"`
}

func (app *application) merge() *cobra.Command {
	var composition composition
	var options mailing
	command := &cobra.Command{
		Use:   "merge <csv>",
		Short: "Open one draft per row of a CSV file",
		Long: "Open one draft in Mail per row of a CSV file. The first line names the columns, and --to, --cc, --subject and the body " +
			"may use {{ column }} placeholders. Every row is checked before any draft opens. Merge never sends.",
		Example: "  mailslot merge people.csv --to '{{email}}' --subject 'Hello {{name}}' --body-file note.txt --dry-run",
		Args:    cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, arguments []string) error {
			drafts, rows, err := options.drafts(arguments[0], &composition, command.InOrStdin())
			if err != nil {
				return err
			}
			account, err := app.resolve(command.Context())
			if err != nil {
				return err
			}
			if err := outgoing(account); err != nil {
				return err
			}
			if err := composition.sender(account); err != nil {
				return err
			}
			output := command.OutOrStdout()
			if options.dryRun {
				return app.mailed(output, rows, true)
			}
			for index := range drafts {
				drafts[index].Account = account.Name
				if _, err := app.backend.Compose(command.Context(), drafts[index]); err != nil {
					if printed := app.mailed(output, rows[:index], false); printed != nil {
						return printed
					}
					return err
				}
			}
			for index := range rows {
				rows[index].Body = ""
			}
			return app.mailed(output, rows, false)
		},
	}
	command.Flags().StringArrayVar(&options.to, "to", nil, "recipient template, such as '{{email}}' (repeatable)")
	command.Flags().StringArrayVar(&options.cc, "cc", nil, "copied recipient template (repeatable)")
	command.Flags().StringVar(&options.subject, "subject", "", "subject line template")
	command.Flags().StringVar(&composition.body, "body", "", "message text template")
	command.Flags().StringVar(&composition.bodyFile, "body-file", "", "file holding the message text template (- for standard input)")
	command.Flags().StringArrayVar(&composition.attachments, "attach", nil, "file to attach to every draft (repeatable)")
	command.Flags().StringVar(&composition.from, "from", "", "sender address; it must belong to the account")
	command.Flags().StringVar(&composition.signature, "signature", "", "name of a Mail signature to append")
	command.MarkFlagsMutuallyExclusive("body", "body-file")
	_ = command.MarkFlagRequired("to")
	_ = command.MarkFlagRequired("subject")
	dryRunFlag(command, &options.dryRun)
	return changing(command)
}

// drafts reads the CSV file and fills every template for every row, so a mistake anywhere opens nothing.
func (options mailing) drafts(path string, composition *composition, input io.Reader) ([]mail.Draft, []merged, error) {
	text, err := composition.text(input)
	if err != nil {
		return nil, nil, err
	}
	files, err := composition.files()
	if err != nil {
		return nil, nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, failure.Wrap(failure.Usage, fmt.Errorf("open CSV file: %w", err), "")
	}
	defer file.Close()
	rows, header, err := merge.Read(file)
	if err != nil {
		return nil, nil, failure.Wrap(failure.Usage, err, "")
	}
	if len(rows) == 0 {
		return nil, nil, failure.New(failure.Usage, fmt.Sprintf("%s has a header but no rows", path), "")
	}

	to := parseAll(options.to)
	cc := parseAll(options.cc)
	subject := merge.Parse(options.subject)
	body := merge.Parse(text)
	if err := known(header, slices.Concat(to, cc, []merge.Template{subject, body})); err != nil {
		return nil, nil, err
	}

	drafts := make([]mail.Draft, 0, len(rows))
	report := make([]merged, 0, len(rows))
	for index, row := range rows {
		number := index + 1
		recipients, err := fillAll(to, row)
		if err != nil {
			return nil, nil, err
		}
		if len(recipients) == 0 {
			return nil, nil, failure.New(failure.Usage, fmt.Sprintf("row %d has no recipient once --to is filled", number), "check that column for empty values")
		}
		copied, err := fillAll(cc, row)
		if err != nil {
			return nil, nil, err
		}
		filledSubject, err := subject.Fill(row)
		if err != nil {
			return nil, nil, failure.Wrap(failure.Usage, err, "")
		}
		filledBody, err := body.Fill(row)
		if err != nil {
			return nil, nil, failure.Wrap(failure.Usage, err, "")
		}
		drafts = append(drafts, mail.Draft{
			From:        composition.from,
			Signature:   composition.signature,
			To:          recipients,
			Cc:          copied,
			Subject:     filledSubject,
			Body:        filledBody,
			Attachments: files,
		})
		report = append(report, merged{Row: number, To: recipients, Subject: filledSubject, Body: filledBody})
	}
	return drafts, report, nil
}

// known refuses any placeholder that names a column the header lacks.
func known(header []string, templates []merge.Template) error {
	columns := make(map[string]bool, len(header))
	for _, name := range header {
		columns[name] = true
	}
	for _, template := range templates {
		for _, field := range template.Fields() {
			if !columns[field] {
				return failure.New(failure.Usage, fmt.Sprintf("the CSV file has no column named %q", field), "columns: "+strings.Join(header, ", "))
			}
		}
	}
	return nil
}

func parseAll(texts []string) []merge.Template {
	templates := make([]merge.Template, 0, len(texts))
	for _, text := range texts {
		templates = append(templates, merge.Parse(text))
	}
	return templates
}

// fillAll fills each recipient template and leaves out the ones that come out empty.
func fillAll(templates []merge.Template, row merge.Row) ([]string, error) {
	values := []string{}
	for _, template := range templates {
		value, err := template.Fill(row)
		if err != nil {
			return nil, failure.Wrap(failure.Usage, err, "")
		}
		if value = strings.TrimSpace(value); value != "" {
			values = append(values, value)
		}
	}
	return values, nil
}

func (app *application) mailed(output io.Writer, rows []merged, dryRun bool) error {
	return app.print(output, rows, func(output io.Writer) {
		if dryRun {
			fmt.Fprintln(output, dryRunNotice)
		}
		if len(rows) > 0 {
			columns(output, "ROW\tTO\tSUBJECT", func(writer io.Writer) {
				for _, row := range rows {
					fmt.Fprintf(writer, "%d\t%s\t%s\n", row.Row, strings.Join(row.To, ", "), row.Subject)
				}
			})
		}
		if !dryRun {
			fmt.Fprintf(output, "Opened %d drafts in Mail; nothing was sent.\n", len(rows))
		}
	})
}
