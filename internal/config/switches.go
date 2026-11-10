package config

import (
	"os"
	"strconv"
)

const (
	readOnlyVariable = "MAILSLOT_READ_ONLY"
	noSendVariable   = "MAILSLOT_NO_SEND"
)

// Switches are the safety switches that stop commands from changing or sending mail.
type Switches struct {
	ReadOnly bool
	NoSend   bool
}

// Switches combines the configuration file with the environment; either can turn a switch on.
func (config Config) Switches() Switches {
	return Switches{
		ReadOnly: config.ReadOnly || enabled(readOnlyVariable),
		NoSend:   (config.Send != nil && !*config.Send) || enabled(noSendVariable),
	}
}

// enabled treats a value that is set but not a recognisable boolean as on, so a typo never disarms a switch.
func enabled(variable string) bool {
	value := os.Getenv(variable)
	if value == "" {
		return false
	}
	parsed, err := strconv.ParseBool(value)
	return err != nil || parsed
}
