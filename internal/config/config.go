// Package config stores the user's amail preferences.
package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

const (
	directoryVariable = "XDG_CONFIG_HOME"
	directoryName     = "amail"
	fileName          = "config.toml"
)

// Config is the content of the configuration file.
type Config struct {
	// Account is the Mail account used when none is named on the command line.
	Account string `toml:"account"`
}

// Path returns where the configuration file lives.
func Path() (string, error) {
	base := os.Getenv(directoryVariable)
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, directoryName, fileName), nil
}

// Load reads the configuration file. A missing file yields an empty configuration.
func Load() (Config, error) {
	var config Config
	path, err := Path()
	if err != nil {
		return config, err
	}
	content, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return config, nil
	}
	if err != nil {
		return config, err
	}
	return config, toml.Unmarshal(content, &config)
}

// Save writes the configuration file, creating its directory when needed.
func (config Config) Save() error {
	path, err := Path()
	if err != nil {
		return err
	}
	content, err := toml.Marshal(config)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}
