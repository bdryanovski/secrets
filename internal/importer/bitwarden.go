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

		// Collect meta fields from CSV.
		meta := &models.CredentialMeta{}
		hasMeta := false

		if folder := getCol(row, colMap, "folder"); folder != "" {
			meta.Folder = folder
			hasMeta = true
		}

		if fav := getCol(row, colMap, "favorite"); fav == "1" || strings.EqualFold(fav, "true") {
			meta.Favorite = true
			hasMeta = true
		}

		if totp := getCol(row, colMap, "login_totp"); totp != "" {
			meta.TOTP = totp
			hasMeta = true
		}

		// Bitwarden CSV stores custom fields as "name: value" pairs separated by newlines.
		if fields := getCol(row, colMap, "fields"); fields != "" {
			for _, line := range strings.Split(fields, "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				parts := strings.SplitN(line, ": ", 2)
				name := parts[0]
				value := ""
				if len(parts) == 2 {
					value = parts[1]
				}
				meta.CustomFields = append(meta.CustomFields, models.CustomField{
					Name:  name,
					Value: value,
					Type:  0, // CSV doesn't indicate field type
				})
				hasMeta = true
			}
		}

		if hasMeta {
			c.Meta = meta
		}

		result.Credentials = append(result.Credentials, c)
	}

	return result, nil
}

// bitwardenJSON represents the top-level Bitwarden JSON export structure.
type bitwardenJSON struct {
	Folders []bitwardenFolder `json:"folders"`
	Items   []bitwardenItem   `json:"items"`
}

type bitwardenItem struct {
	Name     string           `json:"name"`
	Notes    string           `json:"notes"`
	Type     int              `json:"type"`
	Favorite bool             `json:"favorite"`
	FolderID *string          `json:"folderId"`
	Login    *bitwardenLogin  `json:"login"`
	Fields   []bitwardenField `json:"fields"`
}

type bitwardenLogin struct {
	Username string         `json:"username"`
	Password string         `json:"password"`
	TOTP     string         `json:"totp"`
	URIs     []bitwardenURI `json:"uris"`
}

type bitwardenURI struct {
	URI string `json:"uri"`
}

type bitwardenField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Type  int    `json:"type"` // 0=text, 1=hidden, 2=boolean
}

type bitwardenFolder struct {
	ID   string `json:"id"`
	Name string `json:"name"`
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

	// Build folder ID -> name lookup.
	folderMap := make(map[string]string)
	for _, f := range export.Folders {
		folderMap[f.ID] = f.Name
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

		// Collect meta fields.
		meta := &models.CredentialMeta{}
		hasMeta := false

		// TOTP seed
		if item.Login.TOTP != "" {
			meta.TOTP = item.Login.TOTP
			hasMeta = true
		}

		// Extra URIs beyond the first
		if len(item.Login.URIs) > 1 {
			for _, u := range item.Login.URIs[1:] {
				if u.URI != "" {
					meta.ExtraURIs = append(meta.ExtraURIs, u.URI)
				}
			}
			if len(meta.ExtraURIs) > 0 {
				hasMeta = true
			}
		}

		// Folder name
		if item.FolderID != nil && *item.FolderID != "" {
			if name, ok := folderMap[*item.FolderID]; ok {
				meta.Folder = name
				hasMeta = true
			}
		}

		// Favorite flag
		if item.Favorite {
			meta.Favorite = true
			hasMeta = true
		}

		// Custom fields
		for _, f := range item.Fields {
			meta.CustomFields = append(meta.CustomFields, models.CustomField{
				Name:  f.Name,
				Value: f.Value,
				Type:  f.Type,
			})
			hasMeta = true
		}

		if hasMeta {
			c.Meta = meta
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
