package importer

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/bdryanovski/secrets/internal/models"
)

// AppleImporter imports credentials from Apple Passwords CSV exports.
type AppleImporter struct{}

// Name returns the importer name.
func (a *AppleImporter) Name() string {
	return "Apple Passwords"
}

// Import reads an Apple Passwords CSV export and returns credentials.
// Apple Passwords CSV format: Title, URL, Username, Password, Notes, OTPAuth
func (a *AppleImporter) Import(filePath string) (*ImportResult, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(records) < 2 {
		return &ImportResult{Total: 0}, nil
	}

	// Find column indices from header.
	header := records[0]
	colMap := make(map[string]int)
	for i, col := range header {
		colMap[strings.ToLower(strings.TrimSpace(col))] = i
	}

	result := &ImportResult{}
	for _, row := range records[1:] {
		result.Total++

		c := models.Credential{
			Name:     getCol(row, colMap, "title"),
			URL:      getCol(row, colMap, "url"),
			Username: getCol(row, colMap, "username"),
			Password: getCol(row, colMap, "password"),
			Notes:    getCol(row, colMap, "notes"),
		}

		if c.Name == "" && c.URL == "" {
			result.Skipped++
			continue
		}

		if c.Name == "" {
			c.Name = c.URL
		}

		result.Credentials = append(result.Credentials, c)
	}

	return result, nil
}
