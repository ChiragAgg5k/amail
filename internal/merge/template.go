// Package merge fills text templates from the rows of a CSV file.
package merge

import (
	"fmt"
	"strings"
)

const (
	opening = "{{"
	closing = "}}"
)

// piece is a run of literal text or a placeholder naming a column.
type piece struct {
	text   string
	column string
}

// Template is text with {{ column }} placeholders. A {{ without a matching }} is kept as literal text.
type Template struct {
	pieces []piece
}

// Parse splits text into literal runs and placeholders; spaces inside a placeholder are trimmed.
func Parse(text string) Template {
	var pieces []piece
	rest := text
	for {
		start := strings.Index(rest, opening)
		if start < 0 {
			break
		}
		length := strings.Index(rest[start+len(opening):], closing)
		if length < 0 {
			break
		}
		if start > 0 {
			pieces = append(pieces, piece{text: rest[:start]})
		}
		column := strings.TrimSpace(rest[start+len(opening) : start+len(opening)+length])
		pieces = append(pieces, piece{column: column})
		rest = rest[start+len(opening)+length+len(closing):]
	}
	if rest != "" {
		pieces = append(pieces, piece{text: rest})
	}
	return Template{pieces: pieces}
}

// Fields lists the columns the template names, each once, in order of first use.
func (template Template) Fields() []string {
	seen := map[string]bool{}
	fields := []string{}
	for _, part := range template.pieces {
		if !part.placeholder() || seen[part.column] {
			continue
		}
		seen[part.column] = true
		fields = append(fields, part.column)
	}
	return fields
}

// Fill replaces every placeholder with the row's value for that column.
func (template Template) Fill(row Row) (string, error) {
	var filled strings.Builder
	for _, part := range template.pieces {
		if !part.placeholder() {
			filled.WriteString(part.text)
			continue
		}
		value, found := row.Value(part.column)
		if !found {
			return "", fmt.Errorf("no column named %q", part.column)
		}
		filled.WriteString(value)
	}
	return filled.String(), nil
}

func (part piece) placeholder() bool {
	return part.text == ""
}
