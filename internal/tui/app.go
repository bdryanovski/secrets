package tui

import (
	"fmt"

	"github.com/bdryanovski/secrets/internal/config"
	"github.com/bdryanovski/secrets/internal/database"
	"github.com/bdryanovski/secrets/internal/tui/views"
	"github.com/bdryanovski/secrets/internal/version"

	tea "github.com/charmbracelet/bubbletea"
)

// Run starts the TUI application.
func Run(cfg *config.Config) error {
	m := views.NewUnlockModel(cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// RunWithDB starts the TUI application with an already-opened database.
func RunWithDB(cfg *config.Config, db *database.DB) error {
	m := views.NewMainModel(cfg, db)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// VersionString returns the app header with version info.
func VersionString() string {
	return fmt.Sprintf("secrets %s", version.String())
}
