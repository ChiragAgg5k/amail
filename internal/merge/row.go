package merge

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"
)

const byteOrderMark = "\ufeff"

// Row is one data line of a CSV file, read against the file's header.
type Row struct {
	columns map[string]int
	values  []string
}

// Value returns the row's value for a column named in the header.
func (row Row) Value(column string) (string, bool) {
	index, found := row.columns[column]
	if !found {
		return "", false
	}
	return row.values[index], true
}

// Read parses CSV with a header line and returns the data rows and the header's column names.
// Every row must have as many fields as the header.
func Read(reader io.Reader) ([]Row, []string, error) {
	parser := csv.NewReader(reader)
	header, err := parser.Read()
	if errors.Is(err, io.EOF) {
		return nil, nil, errors.New("the CSV file is empty; it needs a header line")
	}
	if err != nil {
		return nil, nil, fmt.Errorf("read CSV header: %w", err)
	}

	columns := make(map[string]int, len(header))
	for index, name := range header {
		if index == 0 {
			name = strings.TrimPrefix(name, byteOrderMark)
		}
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, nil, fmt.Errorf("CSV header column %d has no name", index+1)
		}
		if _, duplicate := columns[name]; duplicate {
			return nil, nil, fmt.Errorf("CSV header names column %q twice", name)
		}
		columns[name] = index
		header[index] = name
	}

	var rows []Row
	for {
		values, err := parser.Read()
		if errors.Is(err, io.EOF) {
			return rows, header, nil
		}
		if err != nil {
			return nil, nil, fmt.Errorf("read CSV: %w", err)
		}
		rows = append(rows, Row{columns: columns, values: values})
	}
}
