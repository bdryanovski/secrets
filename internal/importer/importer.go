package importer

import "github.com/bdryanovski/secrets/internal/models"

// ImportResult holds the results of an import operation.
type ImportResult struct {
	Credentials []models.Credential
	EnvSecrets  []models.EnvSecret
	Total       int
	Skipped     int
	Errors      []string
}

// Importer defines the interface for importing from external password managers.
type Importer interface {
	// Name returns the name of the source (e.g., "Bitwarden", "Apple Passwords").
	Name() string

	// Import reads the given file and returns parsed credentials.
	Import(filePath string) (*ImportResult, error)
}
