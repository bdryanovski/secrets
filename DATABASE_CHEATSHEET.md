# Database Cheatsheet

## SQLite / SQLCipher Basics

### Shell Commands

These are dot-commands run inside the `sqlcipher` prompt (not SQL):

```
.help                  Show all available commands
.tables                List all tables
.schema                Show CREATE statements for all tables
.schema credentials    Show CREATE statement for a specific table
.headers on            Show column names in query output
.mode column           Tabular output (readable)
.mode csv              CSV output
.mode json             JSON output
.mode line             One value per line (good for wide rows)
.width 5 30 30 20      Set column widths for column mode
.quit                  Exit sqlcipher
```

### Output Formatting

```
-- Readable table output
.headers on
.mode column
SELECT id, name, url FROM credentials LIMIT 5;

-- One row at a time, vertically (good for inspecting a single record)
.mode line
SELECT * FROM credentials WHERE id = 1;

-- Export to CSV file
.headers on
.mode csv
.output export.csv
SELECT name, url, username FROM credentials;
.output stdout
```

### Inspecting Tables

```sql
-- Show all tables
.tables

-- Show the full schema (all CREATE TABLE statements)
.schema

-- Show schema for one table
.schema credentials

-- List columns of a table with types
PRAGMA table_info(credentials);

-- Show indexes
.indexes
.indexes credentials

-- Count rows in each table
SELECT 'credentials' as tbl, COUNT(*) as rows FROM credentials
UNION ALL SELECT 'env_secrets', COUNT(*) FROM env_secrets
UNION ALL SELECT 'machines', COUNT(*) FROM machines
UNION ALL SELECT 'metadata', COUNT(*) FROM metadata;
```

### Query Basics

```sql
-- Select specific columns
SELECT name, url FROM credentials;

-- Limit results
SELECT name FROM credentials LIMIT 10;

-- Offset (skip first 10, show next 10)
SELECT name FROM credentials LIMIT 10 OFFSET 10;

-- Sort
SELECT name FROM credentials ORDER BY name ASC;
SELECT name FROM credentials ORDER BY created_at DESC;

-- Filter
SELECT name FROM credentials WHERE username = 'admin';
SELECT name FROM credentials WHERE name LIKE '%git%';    -- contains
SELECT name FROM credentials WHERE name LIKE 'G%';       -- starts with
SELECT name FROM credentials WHERE url != '';             -- not empty

-- Multiple conditions
SELECT name FROM credentials WHERE url != '' AND username != '';
SELECT name FROM credentials WHERE name LIKE '%api%' OR notes LIKE '%api%';

-- Count
SELECT COUNT(*) FROM credentials;
SELECT COUNT(*) FROM credentials WHERE url != '';

-- Distinct values
SELECT DISTINCT environment FROM env_secrets;

-- Group and aggregate
SELECT environment, COUNT(*) as total FROM env_secrets GROUP BY environment;
```

### Updating Data

```sql
-- Update a single field
UPDATE credentials SET name = 'New Name' WHERE id = 42;

-- Update multiple fields
UPDATE credentials SET url = 'https://new.url', username = 'newuser' WHERE id = 42;

-- Update with a condition
UPDATE credentials SET notes = 'migrated' WHERE notes = '';

-- Touch updated_at timestamp
UPDATE credentials SET updated_at = datetime('now') WHERE id = 42;
```

### Inserting Data

```sql
-- Insert a new row (password must be app-encrypted, so this is mainly for non-sensitive fields)
INSERT INTO metadata (key, value) VALUES ('last_export', '2025-01-15');

-- Insert or replace
INSERT OR REPLACE INTO metadata (key, value) VALUES ('last_export', '2025-01-16');
```

### Deleting Data

```sql
-- Delete by ID
DELETE FROM credentials WHERE id = 42;

-- Delete by condition
DELETE FROM credentials WHERE name LIKE '%test%';

-- Delete all rows (keep table)
DELETE FROM credentials;
```

### Transactions

```sql
-- Wrap multiple operations in a transaction
BEGIN;
UPDATE credentials SET notes = 'batch update' WHERE notes = '';
DELETE FROM credentials WHERE name = '';
COMMIT;

-- Or roll back if something goes wrong
BEGIN;
DELETE FROM credentials;
ROLLBACK;   -- undo, nothing was deleted
```

### Explain a Query

```sql
-- Show the query plan (how SQLite will execute it)
EXPLAIN QUERY PLAN SELECT * FROM credentials WHERE name LIKE '%github%';

-- Full execution plan (detailed bytecode)
EXPLAIN SELECT * FROM credentials WHERE name LIKE '%github%';
```

## Connect

```bash
# Get your key
secrets dbkey

# Open the database
sqlcipher ~/.config/secrets/database.db
```

```sql
PRAGMA key = "x'<your-hex-key>'";
PRAGMA cipher_page_size = 4096;
```

## Tables

```sql
.tables
-- credentials, env_secrets, machines, metadata
```

## Credentials

```sql
-- List all
SELECT id, name, url, username FROM credentials ORDER BY name;

-- Search by name
SELECT id, name, url, username FROM credentials WHERE name LIKE '%github%';

-- Search by username
SELECT id, name, url, username FROM credentials WHERE username LIKE '%john%';

-- With notes
SELECT id, name, notes FROM credentials WHERE notes != '';

-- With meta (TOTP, custom fields, folder, etc.)
SELECT id, name, meta FROM credentials WHERE meta != '';

-- Count
SELECT COUNT(*) FROM credentials;

-- Recently added
SELECT id, name, created_at FROM credentials ORDER BY created_at DESC LIMIT 10;

-- Recently updated
SELECT id, name, updated_at FROM credentials ORDER BY updated_at DESC LIMIT 10;
```

## Env Secrets

```sql
-- List all
SELECT id, key, environment, description FROM env_secrets ORDER BY key;

-- Filter by environment
SELECT id, key, description FROM env_secrets WHERE environment = 'production';
SELECT id, key, description FROM env_secrets WHERE environment = 'staging';
SELECT id, key, description FROM env_secrets WHERE environment = 'development';

-- Search by key name
SELECT id, key, environment FROM env_secrets WHERE key LIKE '%API%';

-- Count per environment
SELECT environment, COUNT(*) as total FROM env_secrets GROUP BY environment;
```

## Meta Field (JSON)

The `meta` column on `credentials` is a JSON string. Use SQLite JSON functions to query it.

```sql
-- All credentials with TOTP
SELECT id, name, json_extract(meta, '$.totp') as totp
FROM credentials WHERE json_extract(meta, '$.totp') IS NOT NULL;

-- All favorites
SELECT id, name FROM credentials
WHERE json_extract(meta, '$.favorite') = 1;

-- By folder
SELECT id, name, json_extract(meta, '$.folder') as folder
FROM credentials WHERE json_extract(meta, '$.folder') IS NOT NULL;

-- With custom fields
SELECT id, name, json_extract(meta, '$.custom_fields') as fields
FROM credentials WHERE json_extract(meta, '$.custom_fields') IS NOT NULL;

-- Count by folder
SELECT json_extract(meta, '$.folder') as folder, COUNT(*) as total
FROM credentials WHERE meta != '' AND json_extract(meta, '$.folder') IS NOT NULL
GROUP BY folder;
```

## Metadata (App Key-Value Store)

```sql
SELECT * FROM metadata;
```

## Encrypted Fields

The `password` column in `credentials` and `value` column in `env_secrets` are AES-256-GCM encrypted at the application level. They appear as hex strings and cannot be decrypted via SQL.

```sql
-- You will see hex ciphertext, not plaintext
SELECT id, name, password FROM credentials LIMIT 1;

-- Same for env secret values
SELECT id, key, value FROM env_secrets LIMIT 1;
```

Only the app can decrypt these fields using the master password.

## Bulk Operations

```sql
-- Delete all credentials
DELETE FROM credentials;

-- Delete all env secrets
DELETE FROM env_secrets;

-- Delete credentials by name pattern
DELETE FROM credentials WHERE name LIKE '%test%';

-- Delete env secrets for a specific environment
DELETE FROM env_secrets WHERE environment = 'development';
```

## Useful Queries

```sql
-- Duplicates (same name)
SELECT name, COUNT(*) as cnt FROM credentials GROUP BY name HAVING cnt > 1;

-- Credentials without a URL
SELECT id, name, username FROM credentials WHERE url = '';

-- Credentials without a username
SELECT id, name, url FROM credentials WHERE username = '';

-- Env secrets without a description
SELECT id, key, environment FROM env_secrets WHERE description = '';

-- Export names and URLs (plaintext, no passwords)
.mode csv
.headers on
SELECT name, url, username FROM credentials ORDER BY name;
```
