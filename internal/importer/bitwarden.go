package importer

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bdryanovski/secrets/internal/models"
)

// BitwardenImporter imports credentials from Bitwarden exports.
type BitwardenImporter struct{}

// Name returns the importer name.
func (b *BitwardenImporter) Name() string {
	return "Bitwarden"
}

// Import reads a Bitwarden export file (CSV or JSON) and returns credentials.
func (b *BitwardenImporter) Import(filePath string) (*ImportResult, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".csv":
		return b.importCSV(filePath)
	case ".json":
		return b.importJSON(filePath)
	default:
		return nil, fmt.Errorf("unsupported file format: %s (expected .csv or .json)", ext)
	}
}

func (b *BitwardenImporter) importCSV(filePath string) (*ImportResult, error) {
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
			Name:     getCol(row, colMap, "name"),
			URL:      getCol(row, colMap, "login_uri"),
			Username: getCol(row, colMap, "login_username"),
			Password: getCol(row, colMap, "login_password"),
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

// bitwardenJSON represents the top-level Bitwarden JSON export structure.
type bitwardenJSON struct {
	Items []bitwardenItem `json:"items"`
}

type bitwardenItem struct {
	Name  string          `json:"name"`
	Notes string          `json:"notes"`
	Login *bitwardenLogin `json:"login"`
}

type bitwardenLogin struct {
	Username string         `json:"username"`
	Password string         `json:"password"`
	URIs     []bitwardenURI `json:"uris"`
}

type bitwardenURI struct {
	URI string `json:"uri"`
}

func (b *BitwardenImporter) importJSON(filePath string) (*ImportResult, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var export bitwardenJSON
	if err := json.Unmarshal(data, &export); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	result := &ImportResult{}
	for _, item := range export.Items {
		result.Total++

		if item.Login == nil {
			result.Skipped++
			continue
		}

		url := ""
		if len(item.Login.URIs) > 0 {
			url = item.Login.URIs[0].URI
		}

		c := models.Credential{
			Name:     item.Name,
			URL:      url,
			Username: item.Login.Username,
			Password: item.Login.Password,
			Notes:    item.Notes,
		}

		result.Credentials = append(result.Credentials, c)
	}

	return result, nil
}

func getCol(row []string, colMap map[string]int, key string) string {
	idx, ok := colMap[key]
	if !ok || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}
