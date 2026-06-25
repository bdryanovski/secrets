package views

import (
	"fmt"
	"strings"

	"github.com/bdryanovski/secrets/internal/database"
	"github.com/bdryanovski/secrets/internal/models"
	"github.com/bdryanovski/secrets/internal/tui/styles"

	tea "github.com/charmbracelet/bubbletea"
)

// EnvSecretListModel displays a list of environment secrets.
type EnvSecretListModel struct {
	db     *database.DB
	items  []models.EnvSecret
	cursor int
	err    string
	filter string // environment filter (empty = show all)
}

// NewEnvSecretListModel creates a new env secret list view.
func NewEnvSecretListModel(db *database.DB) *EnvSecretListModel {
	return &EnvSecretListModel{db: db}
}

type envSecretsLoadedMsg struct {
	items []models.EnvSecret
	err   error
}

func (m *EnvSecretListModel) loadEnvSecrets() tea.Cmd {
	db := m.db
	filter := m.filter
	return func() tea.Msg {
		items, err := db.ListEnvSecrets(filter)
		return envSecretsLoadedMsg{items: items, err: err}
	}
}

func (m *EnvSecretListModel) update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case envSecretsLoadedMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.items = msg.items
			m.err = ""
			if m.cursor >= len(m.items) {
				m.cursor = max(0, len(m.items)-1)
			}
		}
	}
	return nil
}

func (m *EnvSecretListModel) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}
	case "home":
		m.cursor = 0
	case "end":
		m.cursor = len(m.items) - 1
	case "1":
		m.filter = "development"
		return m.loadEnvSecrets()
	case "2":
		m.filter = "staging"
		return m.loadEnvSecrets()
	case "3":
		m.filter = "production"
		return m.loadEnvSecrets()
	case "0":
		m.filter = ""
		return m.loadEnvSecrets()
	}
	return nil
}

// Selected returns the currently selected env secret, or nil.
func (m *EnvSecretListModel) Selected() *models.EnvSecret {
	if m.cursor >= 0 && m.cursor < len(m.items) {
		return &m.items[m.cursor]
	}
	return nil
}

func (m *EnvSecretListModel) View() string {
	var b strings.Builder

	// Environment filter bar
	b.WriteString("  ")
	envs := []struct{ key, label string }{
		{"", "All (0)"},
		{"development", "Dev (1)"},
		{"staging", "Staging (2)"},
		{"production", "Prod (3)"},
	}
	for _, e := range envs {
		if m.filter == e.key {
			b.WriteString(styles.ActiveTabStyle.Render(e.label))
		} else {
			b.WriteString(styles.InactiveTabStyle.Render(e.label))
		}
	}
	b.WriteString("\n\n")

	if m.err != "" {
		b.WriteString(styles.DangerStyle.Render("  Error: " + m.err))
		b.WriteString("\n")
		return b.String()
	}

	if len(m.items) == 0 {
		b.WriteString(styles.MutedStyle.Render("  No env secrets stored yet. Press 'a' to add one."))
		b.WriteString("\n")
		return b.String()
	}

	for i, env := range m.items {
		cursor := "  "
		if i == m.cursor {
			cursor = styles.SelectedStyle.Render("> ")
		}

		key := env.Key
		if len(key) > 30 {
			key = key[:27] + "..."
		}

		envLabel := envBadge(env.Environment)
		line := fmt.Sprintf("%-32s %s", key, envLabel)

		if i == m.cursor {
			b.WriteString(cursor + styles.SelectedStyle.Render(line))
		} else {
			b.WriteString(cursor + styles.NormalStyle.Render(line))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func envBadge(env string) string {
	switch env {
	case "production":
		return styles.DangerStyle.Render("[prod]")
	case "staging":
		return styles.WarningStyle.Render("[staging]")
	case "development":
		return styles.SuccessStyle.Render("[dev]")
	default:
		return styles.MutedStyle.Render("[" + env + "]")
	}
}
