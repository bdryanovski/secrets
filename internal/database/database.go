package database

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/bdryanovski/secrets/internal/crypto"
	"github.com/bdryanovski/secrets/internal/models"

	_ "github.com/mutecomm/go-sqlcipher/v4"
)

// DB wraps the SQLCipher-encrypted database connection.
type DB struct {
	conn      *sql.DB
	encKey    []byte // AES key for application-level encryption of sensitive fields
	masterKey string // The hex-encoded SQLCipher key
}

// Open opens (or creates) an encrypted SQLite database at the given path.
// The masterPassword is used both to unlock SQLCipher and to derive an
// application-level encryption key for sensitive fields.
func Open(dbPath, masterPassword string) (*DB, error) {
	// Derive the SQLCipher key from the master password.
	// We use a fixed salt derived from the password itself for SQLCipher
	// (SQLCipher handles its own KDF internally, but we pass the key as hex).
	salt := []byte("secrets-sqlcipher-salt-v1")
	key := crypto.DeriveKey(masterPassword, salt)
	hexKey := hex.EncodeToString(key)

	// Derive a separate key for application-level field encryption.
	fieldSalt := []byte("secrets-field-encrypt-v1")
	fieldKey := crypto.DeriveKey(masterPassword, fieldSalt)

	dsn := fmt.Sprintf("%s?_pragma_key=x'%s'&_pragma_cipher_page_size=4096", dbPath, hexKey)

	conn, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify the database can be read (wrong password will fail here).
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to unlock database (wrong password?): %w", err)
	}

	db := &DB{
		conn:      conn,
		encKey:    fieldKey,
		masterKey: hexKey,
	}

	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

// migrate creates the database schema if it does not exist.
func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS credentials (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		name        TEXT NOT NULL,
		url         TEXT NOT NULL DEFAULT '',
		username    TEXT NOT NULL DEFAULT '',
		password    TEXT NOT NULL DEFAULT '',
		notes       TEXT NOT NULL DEFAULT '',
		created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
		updated_at  DATETIME NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS env_secrets (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		key         TEXT NOT NULL,
		value       TEXT NOT NULL DEFAULT '',
		environment TEXT NOT NULL DEFAULT 'development',
		description TEXT NOT NULL DEFAULT '',
		created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
		updated_at  DATETIME NOT NULL DEFAULT (datetime('now')),
		UNIQUE(key, environment)
	);

	CREATE TABLE IF NOT EXISTS machines (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		fingerprint TEXT NOT NULL UNIQUE,
		name        TEXT NOT NULL DEFAULT '',
		public_key  BLOB,
		created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
		last_sync_at DATETIME
	);

	CREATE TABLE IF NOT EXISTS metadata (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	`
	_, err := db.conn.Exec(schema)
	return err
}

// --- Credential CRUD ---

// CreateCredential inserts a new credential. The password is encrypted at the application level.
func (db *DB) CreateCredential(c *models.Credential) error {
	encPassword, err := crypto.Encrypt(c.Password, db.encKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt password: %w", err)
	}

	result, err := db.conn.Exec(
		`INSERT INTO credentials (name, url, username, password, notes, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		c.Name, c.URL, c.Username, encPassword, c.Notes, time.Now(), time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to insert credential: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	c.ID = id
	return nil
}

// GetCredential retrieves a credential by ID. The password is decrypted.
func (db *DB) GetCredential(id int64) (*models.Credential, error) {
	c := &models.Credential{}
	var encPassword string
	err := db.conn.QueryRow(
		`SELECT id, name, url, username, password, notes, created_at, updated_at
		 FROM credentials WHERE id = ?`, id,
	).Scan(&c.ID, &c.Name, &c.URL, &c.Username, &encPassword, &c.Notes, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get credential: %w", err)
	}

	c.Password, err = crypto.Decrypt(encPassword, db.encKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt password: %w", err)
	}
	return c, nil
}

// ListCredentials returns all credentials. Passwords are NOT decrypted in the list view.
func (db *DB) ListCredentials() ([]models.Credential, error) {
	rows, err := db.conn.Query(
		`SELECT id, name, url, username, notes, created_at, updated_at
		 FROM credentials ORDER BY name ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list credentials: %w", err)
	}
	defer rows.Close()

	var creds []models.Credential
	for rows.Next() {
		var c models.Credential
		if err := rows.Scan(&c.ID, &c.Name, &c.URL, &c.Username, &c.Notes, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		creds = append(creds, c)
	}
	return creds, rows.Err()
}

// UpdateCredential updates an existing credential.
func (db *DB) UpdateCredential(c *models.Credential) error {
	encPassword, err := crypto.Encrypt(c.Password, db.encKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt password: %w", err)
	}

	_, err = db.conn.Exec(
		`UPDATE credentials SET name = ?, url = ?, username = ?, password = ?, notes = ?, updated_at = ?
		 WHERE id = ?`,
		c.Name, c.URL, c.Username, encPassword, c.Notes, time.Now(), c.ID,
	)
	return err
}

// DeleteCredential removes a credential by ID.
func (db *DB) DeleteCredential(id int64) error {
	_, err := db.conn.Exec(`DELETE FROM credentials WHERE id = ?`, id)
	return err
}

// SearchCredentials searches credentials by name or username.
func (db *DB) SearchCredentials(query string) ([]models.Credential, error) {
	like := "%" + query + "%"
	rows, err := db.conn.Query(
		`SELECT id, name, url, username, notes, created_at, updated_at
		 FROM credentials WHERE name LIKE ? OR username LIKE ? OR url LIKE ?
		 ORDER BY name ASC`,
		like, like, like,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var creds []models.Credential
	for rows.Next() {
		var c models.Credential
		if err := rows.Scan(&c.ID, &c.Name, &c.URL, &c.Username, &c.Notes, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		creds = append(creds, c)
	}
	return creds, rows.Err()
}

// --- EnvSecret CRUD ---

// CreateEnvSecret inserts a new environment secret. The value is encrypted.
func (db *DB) CreateEnvSecret(e *models.EnvSecret) error {
	encValue, err := crypto.Encrypt(e.Value, db.encKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt env value: %w", err)
	}

	result, err := db.conn.Exec(
		`INSERT INTO env_secrets (key, value, environment, description, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		e.Key, encValue, e.Environment, e.Description, time.Now(), time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to insert env secret: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	e.ID = id
	return nil
}

// GetEnvSecret retrieves an env secret by ID with decrypted value.
func (db *DB) GetEnvSecret(id int64) (*models.EnvSecret, error) {
	e := &models.EnvSecret{}
	var encValue string
	err := db.conn.QueryRow(
		`SELECT id, key, value, environment, description, created_at, updated_at
		 FROM env_secrets WHERE id = ?`, id,
	).Scan(&e.ID, &e.Key, &encValue, &e.Environment, &e.Description, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get env secret: %w", err)
	}

	e.Value, err = crypto.Decrypt(encValue, db.encKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt env value: %w", err)
	}
	return e, nil
}

// GetEnvSecretByKeyEnv retrieves an env secret by key and environment with decrypted value.
func (db *DB) GetEnvSecretByKeyEnv(key, environment string) (*models.EnvSecret, error) {
	e := &models.EnvSecret{}
	var encValue string
	err := db.conn.QueryRow(
		`SELECT id, key, value, environment, description, created_at, updated_at
		 FROM env_secrets WHERE key = ? AND environment = ?`, key, environment,
	).Scan(&e.ID, &e.Key, &encValue, &e.Environment, &e.Description, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get env secret: %w", err)
	}

	e.Value, err = crypto.Decrypt(encValue, db.encKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt env value: %w", err)
	}
	return e, nil
}

// ListEnvSecrets returns all env secrets, optionally filtered by environment.
// Values are NOT decrypted in the list view.
func (db *DB) ListEnvSecrets(environment string) ([]models.EnvSecret, error) {
	var rows *sql.Rows
	var err error

	if environment == "" {
		rows, err = db.conn.Query(
			`SELECT id, key, environment, description, created_at, updated_at
			 FROM env_secrets ORDER BY key ASC, environment ASC`,
		)
	} else {
		rows, err = db.conn.Query(
			`SELECT id, key, environment, description, created_at, updated_at
			 FROM env_secrets WHERE environment = ? ORDER BY key ASC`,
			environment,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list env secrets: %w", err)
	}
	defer rows.Close()

	var secrets []models.EnvSecret
	for rows.Next() {
		var e models.EnvSecret
		if err := rows.Scan(&e.ID, &e.Key, &e.Environment, &e.Description, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		secrets = append(secrets, e)
	}
	return secrets, rows.Err()
}

// ListEnvSecretsDecrypted returns all env secrets for a given environment with decrypted values.
func (db *DB) ListEnvSecretsDecrypted(environment string) ([]models.EnvSecret, error) {
	rows, err := db.conn.Query(
		`SELECT id, key, value, environment, description, created_at, updated_at
		 FROM env_secrets WHERE environment = ? ORDER BY key ASC`,
		environment,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list env secrets: %w", err)
	}
	defer rows.Close()

	var secrets []models.EnvSecret
	for rows.Next() {
		var e models.EnvSecret
		var encValue string
		if err := rows.Scan(&e.ID, &e.Key, &encValue, &e.Environment, &e.Description, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		e.Value, err = crypto.Decrypt(encValue, db.encKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt value for key %s: %w", e.Key, err)
		}
		secrets = append(secrets, e)
	}
	return secrets, rows.Err()
}

// UpdateEnvSecret updates an existing env secret.
func (db *DB) UpdateEnvSecret(e *models.EnvSecret) error {
	encValue, err := crypto.Encrypt(e.Value, db.encKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt env value: %w", err)
	}

	_, err = db.conn.Exec(
		`UPDATE env_secrets SET key = ?, value = ?, environment = ?, description = ?, updated_at = ?
		 WHERE id = ?`,
		e.Key, encValue, e.Environment, e.Description, time.Now(), e.ID,
	)
	return err
}

// DeleteEnvSecret removes an env secret by ID.
func (db *DB) DeleteEnvSecret(id int64) error {
	_, err := db.conn.Exec(`DELETE FROM env_secrets WHERE id = ?`, id)
	return err
}

// SearchEnvSecrets searches env secrets by key or description.
func (db *DB) SearchEnvSecrets(query string) ([]models.EnvSecret, error) {
	like := "%" + query + "%"
	rows, err := db.conn.Query(
		`SELECT id, key, environment, description, created_at, updated_at
		 FROM env_secrets WHERE key LIKE ? OR description LIKE ?
		 ORDER BY key ASC, environment ASC`,
		like, like,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var secrets []models.EnvSecret
	for rows.Next() {
		var e models.EnvSecret
		if err := rows.Scan(&e.ID, &e.Key, &e.Environment, &e.Description, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		secrets = append(secrets, e)
	}
	return secrets, rows.Err()
}

// --- Metadata helpers ---

// SetMeta stores a key-value pair in the metadata table.
func (db *DB) SetMeta(key, value string) error {
	_, err := db.conn.Exec(
		`INSERT INTO metadata (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value,
	)
	return err
}

// GetMeta retrieves a value from the metadata table.
func (db *DB) GetMeta(key string) (string, error) {
	var value string
	err := db.conn.QueryRow(`SELECT value FROM metadata WHERE key = ?`, key).Scan(&value)
	if err != nil {
		return "", err
	}
	return value, nil
}
