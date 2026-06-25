package config

import (
	"os"
	"path/filepath"
	"time"
)

const (
	// AppName is the name of the application.
	AppName = "secrets"

	// DefaultDBFile is the default database filename.
	DefaultDBFile = "database.db"

	// DefaultConfigDir is the directory name under ~/.config/.
	DefaultConfigDir = "secrets"

	// DefaultClipboardTimeout is how long a password stays in the clipboard.
	DefaultClipboardTimeout = 1 * time.Minute

	// DefaultPasswordLength is the default generated password length.
	DefaultPasswordLength = 24

	// DefaultSyncPort is the default port for machine-to-machine sync.
	DefaultSyncPort = 9876
)

// Config holds the application configuration.
type Config struct {
	// ConfigDir is the path to the configuration directory.
	ConfigDir string

	// DBPath is the full path to the encrypted database file.
	DBPath string

	// ClipboardTimeout is how long a copied password stays in the clipboard.
	ClipboardTimeout time.Duration

	// PasswordLength is the default length for generated passwords.
	PasswordLength int

	// SyncPort is the port used for machine-to-machine sync.
	SyncPort int
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configDir := filepath.Join(homeDir, ".config", DefaultConfigDir)

	return &Config{
		ConfigDir:        configDir,
		DBPath:           filepath.Join(configDir, DefaultDBFile),
		ClipboardTimeout: DefaultClipboardTimeout,
		PasswordLength:   DefaultPasswordLength,
		SyncPort:         DefaultSyncPort,
	}, nil
}

// EnsureConfigDir creates the configuration directory if it does not exist.
func (c *Config) EnsureConfigDir() error {
	return os.MkdirAll(c.ConfigDir, 0700)
}
