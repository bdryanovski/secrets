package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"syscall"

	"golang.org/x/term"

	"github.com/bdryanovski/secrets/internal/config"
	"github.com/bdryanovski/secrets/internal/crypto"
	"github.com/bdryanovski/secrets/internal/database"
	"github.com/bdryanovski/secrets/internal/tui"
	"github.com/bdryanovski/secrets/internal/version"
)

// promptPassword reads a password from the terminal without echoing it.
func promptPassword() string {
	fmt.Fprint(os.Stderr, "Master password: ")
	pw, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Fprintln(os.Stderr) // newline after hidden input
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading password: %s\n", err)
		os.Exit(1)
	}
	return strings.TrimSpace(string(pw))
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "--version", "-v":
			fmt.Println(version.String())
			return
		case "env":
			handleEnvCommand(os.Args[2:])
			return
		case "purge":
			handlePurgeCommand(os.Args[2:])
			return
		case "dbkey":
			handleDBKeyCommand(os.Args[2:])
			return
		case "help", "--help", "-h":
			printHelp()
			return
		}
	}

	cfg, err := config.DefaultConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %s\n", err)
		os.Exit(1)
	}

	if err := tui.Run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

// handleEnvCommand outputs env secrets for shell sourcing.
// Usage: secrets env --profile <environment> [--password <password>]
func handleEnvCommand(args []string) {
	profile := "development"
	password := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--profile", "-p":
			if i+1 < len(args) {
				profile = args[i+1]
				i++
			}
		case "--password":
			if i+1 < len(args) {
				password = args[i+1]
				i++
			}
		}
	}

	if password == "" {
		password = promptPassword()
	}

	cfg, err := config.DefaultConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	db, err := database.Open(cfg.DBPath, password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to unlock vault: %s\n", err)
		os.Exit(1)
	}
	defer db.Close()

	secrets, err := db.ListEnvSecretsDecrypted(profile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading secrets: %s\n", err)
		os.Exit(1)
	}

	// Output in shell-sourceable format: export KEY="value"
	for _, s := range secrets {
		// Escape double quotes and backslashes in value.
		escaped := strings.ReplaceAll(s.Value, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		fmt.Printf("export %s=\"%s\"\n", s.Key, escaped)
	}
}

// handlePurgeCommand deletes all data from the vault.
// Usage: secrets purge [--password <password>]
func handlePurgeCommand(args []string) {
	password := ""

	for i := 0; i < len(args); i++ {
		if args[i] == "--password" && i+1 < len(args) {
			password = args[i+1]
			i++
		}
	}

	if password == "" {
		password = promptPassword()
	}

	cfg, err := config.DefaultConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	db, err := database.Open(cfg.DBPath, password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to unlock vault: %s\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.PurgeAll(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to purge vault: %s\n", err)
		os.Exit(1)
	}

	fmt.Println("Vault purged. All credentials and env secrets have been deleted.")
}

// handleDBKeyCommand derives and prints the SQLCipher hex key for direct database access.
// Usage: secrets dbkey [--password <password>]
func handleDBKeyCommand(args []string) {
	password := ""

	for i := 0; i < len(args); i++ {
		if args[i] == "--password" && i+1 < len(args) {
			password = args[i+1]
			i++
		}
	}

	if password == "" {
		password = promptPassword()
	}

	cfg, err := config.DefaultConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	salt := []byte("secrets-sqlcipher-salt-v1")
	key := crypto.DeriveKey(password, salt)
	hexKey := hex.EncodeToString(key)

	fmt.Println("Database path:")
	fmt.Println("  " + cfg.DBPath)
	fmt.Println()
	fmt.Println("SQLCipher hex key:")
	fmt.Println("  " + hexKey)
	fmt.Println()
	fmt.Println("Connect with sqlcipher:")
	fmt.Printf("  sqlcipher %s\n", cfg.DBPath)
	fmt.Printf("  PRAGMA key = \"x'%s'\";\n", hexKey)
	fmt.Println("  PRAGMA cipher_page_size = 4096;")
	fmt.Println("  .tables")
	fmt.Println()
	fmt.Println("Note: password and value fields are additionally encrypted with AES-256-GCM")
	fmt.Println("and will appear as hex-encoded ciphertext in query results.")
}

func printHelp() {
	fmt.Println("secrets - A local secrets manager")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  secrets              Launch the TUI")
	fmt.Println("  secrets env          Output env secrets for shell sourcing")
	fmt.Println("    --profile, -p      Environment profile (default: development)")
	fmt.Println("    --password         Master password (will prompt if not given)")
	fmt.Println("  secrets purge        Delete all data from the vault")
	fmt.Println("    --password         Master password (will prompt if not given)")
	fmt.Println("  secrets dbkey        Print the SQLCipher key for direct DB access")
	fmt.Println("    --password         Master password (will prompt if not given)")
	fmt.Println("  secrets version      Show version")
	fmt.Println("  secrets help         Show this help")
	fmt.Println()
	fmt.Println("Shell integration:")
	fmt.Println("  eval $(secrets env --profile production --password <pw>)")
	fmt.Println()
	fmt.Println(version.String())
}
