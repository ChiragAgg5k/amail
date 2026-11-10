package cli

import (
	"fmt"
	"io"

	"github.com/ChiragAgg5k/mailslot/internal/failure"
)

// diagnosis is everything doctor found, in the order it checked.
type diagnosis struct {
	Checks []check `json:"checks"`
}

func (found diagnosis) table(output io.Writer) {
	columns(output, "CHECK\tSTATUS\tDETAIL", func(writer io.Writer) {
		for _, item := range found.Checks {
			fmt.Fprintf(writer, "%s\t%s\t%s\n", item.Name, item.Status, item.Detail)
			if item.Hint != "" && (item.Status == checkFailed || item.Status == checkWarned) {
				fmt.Fprintf(writer, "\t\thint: %s\n", item.Hint)
			}
		}
	})
}

// verdict fails with the exit code of the first failed check, so a missing permission exits 3.
func (found diagnosis) verdict() error {
	for _, item := range found.Checks {
		if item.Status == checkFailed {
			return failure.New(item.code, fmt.Sprintf("the %s check failed", item.Name), "")
		}
	}
	return nil
}
