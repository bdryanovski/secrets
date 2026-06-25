package models

import "time"

// CredentialMeta holds extra structured data imported from external sources.
type CredentialMeta struct {
	TOTP         string            `json:"totp,omitempty"`          // TOTP authenticator seed
	ExtraURIs    []string          `json:"extra_uris,omitempty"`    // Additional login URIs beyond the primary
	Folder       string            `json:"folder,omitempty"`        // Source folder/category name
	Favorite     bool              `json:"favorite,omitempty"`      // Whether this was marked as a favorite
	CustomFields []CustomField     `json:"custom_fields,omitempty"` // Arbitrary key-value custom fields
	Extra        map[string]string `json:"extra,omitempty"`         // Catch-all for any other data
}

// CustomField represents an arbitrary key-value field on a credential.
type CustomField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Type  int    `json:"type"` // 0=text, 1=hidden, 2=boolean
}

// Credential represents a website login entry (username + password).
type Credential struct {
	ID        int64           `json:"id"`
	Name      string          `json:"name"`     // Friendly name (e.g., "GitHub")
	URL       string          `json:"url"`      // Website URL
	Username  string          `json:"username"` // Login username or email
	Password  string          `json:"password"` // Encrypted password
	Notes     string          `json:"notes"`    // Optional notes
	Meta      *CredentialMeta `json:"meta"`     // Extra structured metadata (TOTP, custom fields, etc.)
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// EnvSecret represents an environment variable secret with multi-environment support.
type EnvSecret struct {
	ID          int64     `json:"id"`
	Key         string    `json:"key"`         // Environment variable name (e.g., "API_KEY")
	Value       string    `json:"value"`       // Encrypted value
	Environment string    `json:"environment"` // Environment profile (e.g., "production", "staging", "development")
	Description string    `json:"description"` // Optional description
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// MachineInfo represents a registered machine for sync.
type MachineInfo struct {
	ID          int64     `json:"id"`
	Fingerprint string    `json:"fingerprint"` // Unique machine identifier
	Name        string    `json:"name"`        // Human-readable machine name
	PublicKey   []byte    `json:"public_key"`  // Public key for encrypted sync
	CreatedAt   time.Time `json:"created_at"`
	LastSyncAt  time.Time `json:"last_sync_at"`
}

// EnvironmentList is a list of standard environment profiles.
var EnvironmentList = []string{
	"development",
	"staging",
	"production",
}
