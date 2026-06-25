package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/bdryanovski/secrets/internal/config"
	"github.com/bdryanovski/secrets/internal/database"
	"github.com/bdryanovski/secrets/internal/tui"
	"github.com/bdryanovski/secrets/internal/version"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "--version", "-v":
			fmt.Println(version.String())
			return
		case "env":
			handleEnvCommand(os.Args[2:])
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
		fmt.Fprint(os.Stderr, "Master password: ")
		// Read password from stdin (simple, non-echo version).
		var pw string
		fmt.Scanln(&pw)
		password = strings.TrimSpace(pw)
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

func printHelp() {
	fmt.Println("secrets - A local secrets manager")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  secrets              Launch the TUI")
	fmt.Println("  secrets env          Output env secrets for shell sourcing")
	fmt.Println("    --profile, -p      Environment profile (default: development)")
	fmt.Println("    --password         Master password (will prompt if not given)")
	fmt.Println("  secrets version      Show version")
	fmt.Println("  secrets help         Show this help")
	fmt.Println()
	fmt.Println("Shell integration:")
	fmt.Println("  eval $(secrets env --profile production --password <pw>)")
	fmt.Println()
	fmt.Println(version.String())
}
