# Developer Guide

## Project Structure

```
secrets/
  cmd/secrets/main.go           Entry point, CLI commands
  internal/
    config/config.go             App configuration
    crypto/crypto.go             Argon2id key derivation, AES-256-GCM encryption
    database/database.go         SQLCipher database layer, CRUD operations
    models/models.go             Data models (Credential, EnvSecret, etc.)
    importer/                    Bitwarden & Apple import
    clipboard/clipboard.go       Clipboard with auto-clear
    password/generator.go        Random password generation
    sync/                        Machine-to-machine sync
    tui/
      app.go                     TUI launcher
      views/                     Bubbletea views (unlock, main, lists, forms, detail)
      styles/styles.go           Color palette and component styles
    version/version.go           Build-time version info
```

## Building

```bash
make build          # Build for current platform
make build-linux    # Build for Linux
make build-darwin   # Build for macOS
make clean          # Remove build artifacts
```

Or directly with Go:

```bash
go build -o secrets ./cmd/secrets/
```

## CLI Commands

```
secrets                Launch the TUI
secrets env            Output env secrets for shell sourcing
secrets purge          Delete all data from the vault
secrets dbkey          Print the SQLCipher key for direct DB access
secrets version        Show version
secrets help           Show help
```

## Master Password

The master password is the single secret that protects all data in the vault. It is never stored anywhere -- not on disk, not in memory after use, and not in any config file. Instead, it is used at runtime to derive encryption keys.

### How it works

1. **First launch** -- when no database file exists at `~/.config/secrets/database.db`, the TUI presents a two-step password creation flow:
   - Enter a master password (minimum 8 characters)
   - Confirm the password (must match exactly)
   - The password is passed to `database.Open()` which creates a new encrypted database

2. **Subsequent launches** -- the TUI prompts for the master password. The password is passed to `database.Open()` which attempts to unlock the existing database. If the password is wrong, SQLCipher fails to read the database and returns an error.

3. **CLI commands** (`env`, `purge`, `dbkey`) -- the password is provided via `--password` flag or prompted interactively (input is hidden, not echoed to the terminal).

### What happens with the password

The master password is never stored. It is used to derive two separate 256-bit keys via Argon2id, then discarded:

```
master password
    |
    +-- Argon2id(salt: "secrets-sqlcipher-salt-v1")  --> SQLCipher key (32 bytes)
    |       Used as PRAGMA key to encrypt/decrypt the entire database file.
    |       Passed as hex to SQLCipher via the connection DSN.
    |
    +-- Argon2id(salt: "secrets-field-encrypt-v1")   --> Field encryption key (32 bytes)
            Used for AES-256-GCM encryption of the `password` and `value`
            columns within the already-encrypted database.
```

Using two different salts ensures the two derived keys are completely independent. Knowing one does not reveal the other.

### Why two layers

- **SQLCipher** encrypts the entire database file. Without the correct key, the file looks like random bytes. This protects against someone copying the database file.
- **AES-256-GCM field encryption** adds a second layer on the most sensitive fields (passwords and secret values). Even if someone extracts the SQLCipher key and queries the database directly, they still cannot read the actual passwords -- those remain encrypted with the field key.

### Password change

There is currently no password change feature. To change the master password, you would need to:
1. Export your data (or note it down)
2. Delete the database file (`~/.config/secrets/database.db`)
3. Relaunch the app, which will prompt you to create a new password
4. Re-import your data

### Security properties

- The master password is never written to disk
- The master password is never logged or transmitted
- Key derivation uses Argon2id (memory-hard, resistant to GPU/ASIC attacks)
- Each key derivation requires 64 MB of memory and 3 iterations
- The derived keys exist only in process memory while the app is running
- CLI password input is hidden (not echoed to the terminal)

## Database Encryption

The database uses two layers of encryption:

1. **SQLCipher** -- encrypts the entire SQLite database file. The key is derived from the master password using Argon2id with a fixed salt (`secrets-sqlcipher-salt-v1`).

2. **AES-256-GCM** -- additionally encrypts sensitive fields (`password` on credentials, `value` on env secrets) at the application level. Uses a separate key derived with salt `secrets-field-encrypt-v1`.

This means even if someone gains access to the SQLCipher key, the passwords and secret values remain encrypted.

### Key Derivation Parameters

- Algorithm: Argon2id
- Time: 3 iterations
- Memory: 64 MB
- Threads: 4
- Key length: 32 bytes (AES-256)

## Connecting to the Database Directly

You need `sqlcipher`, not regular `sqlite3`. Regular `sqlite3` will report the file as corrupted.

### Install sqlcipher

**Arch Linux:**

```bash
sudo pacman -S sqlcipher
```

**macOS:**

```bash
brew install sqlcipher
```

### Get your database key

```bash
secrets dbkey
# or non-interactively:
secrets dbkey --password <your-master-password>
```

This prints the hex-encoded SQLCipher key and the exact commands to connect.

### Connect

```bash
sqlcipher ~/.config/secrets/database.db
```

Then inside the sqlcipher prompt:

```sql
PRAGMA key = "x'<hex key from dbkey output>'";
PRAGMA cipher_page_size = 4096;

-- Verify it works
.tables

-- Browse credentials (passwords are AES-encrypted, shown as hex)
SELECT id, name, url, username, notes FROM credentials;

-- Browse env secrets (values are AES-encrypted, shown as hex)
SELECT id, key, environment, description FROM env_secrets;

-- Browse metadata
SELECT * FROM metadata;
```

The `password` column in `credentials` and `value` column in `env_secrets` will appear as hex-encoded ciphertext. These are encrypted with AES-256-GCM at the application level and can only be decrypted by the app using the master password.

### Database Schema

```sql
credentials (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL,
    url         TEXT NOT NULL DEFAULT '',
    username    TEXT NOT NULL DEFAULT '',
    password    TEXT NOT NULL DEFAULT '',       -- AES-256-GCM encrypted
    notes       TEXT NOT NULL DEFAULT '',
    meta        TEXT NOT NULL DEFAULT '',       -- JSON (custom fields, TOTP, etc.)
    created_at  DATETIME,
    updated_at  DATETIME
);

env_secrets (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    key         TEXT NOT NULL,
    value       TEXT NOT NULL DEFAULT '',       -- AES-256-GCM encrypted
    environment TEXT NOT NULL DEFAULT 'development',
    description TEXT NOT NULL DEFAULT '',
    created_at  DATETIME,
    updated_at  DATETIME,
    UNIQUE(key, environment)
);

machines (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    fingerprint  TEXT NOT NULL UNIQUE,
    name         TEXT NOT NULL DEFAULT '',
    public_key   BLOB,
    created_at   DATETIME,
    last_sync_at DATETIME
);

metadata (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
```

## Purging Data

To delete all credentials and env secrets from the vault (useful for testing imports):

```bash
secrets purge
# or non-interactively:
secrets purge --password <your-master-password>
```

This removes all data but keeps the database file and its encryption intact.
