package mail

import (
	"fmt"
	"strings"

	"github.com/ChiragAgg5k/mailslot/internal/failure"
)

// FlagColor is Mail's flag index.
//
// Mail's dictionary documents flag index only as -1 for none; the colour order follows Mail's Flag menu
// as reported by the scripting community and needs confirming in the e2e suite.
type FlagColor int

const (
	Clear  FlagColor = -1
	Red    FlagColor = 0
	Orange FlagColor = 1
	Yellow FlagColor = 2
	Green  FlagColor = 3
	Blue   FlagColor = 4
	Purple FlagColor = 5
	Gray   FlagColor = 6
)

const clearName = "none"

var colorNames = []string{"red", "orange", "yellow", "green", "blue", "purple", "gray"}

// ParseFlagColor reads a colour name such as red or blue, ignoring case.
func ParseFlagColor(name string) (FlagColor, error) {
	wanted := strings.ToLower(strings.TrimSpace(name))
	for index, candidate := range colorNames {
		if candidate == wanted {
			return FlagColor(index), nil
		}
	}
	return Clear, failure.New(failure.Usage, fmt.Sprintf("%q is not a flag colour", name), "use one of "+strings.Join(colorNames, ", "))
}

func (color FlagColor) String() string {
	if color < Red || int(color) >= len(colorNames) {
		return clearName
	}
	return colorNames[color]
}
