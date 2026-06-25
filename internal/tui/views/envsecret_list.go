package views

import (
	"fmt"
	"strings"

	"github.com/bdryanovski/secrets/internal/database"
	"github.com/bdryanovski/secrets/internal/models"
	"github.com/bdryanovski/secrets/internal/tui/styles"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// EnvSecretListModel displays a list of environment secrets.
type EnvSecretListModel struct {
	db     *database.DB
	items  []models.EnvSecret
	cursor int
	err    string
	filter string
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
		if len(m.items) > 0 {
			m.cursor = len(m.items) - 1
		}
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

	// Environment filter pills
	b.WriteString("\n")
	b.WriteString("  ")
	filters := []struct {
		key, label, value string
	}{
		{"0", "All", ""},
		{"1", "Dev", "development"},
		{"2", "Staging", "staging"},
		{"3", "Prod", "production"},
	}
	for i, f := range filters {
		if i > 0 {
			b.WriteString(" ")
		}
		label := f.label + " (" + f.key + ")"
		if m.filter == f.value {
			b.WriteString(styles.Badge(" "+label+" ", styles.BgDark, styles.Primary))
		} else {
			b.WriteString(styles.Badge(" "+label+" ", styles.TextDim, styles.BgCard))
		}
	}
	b.WriteString("\n")

	if m.err != "" {
		b.WriteString("\n")
		b.WriteString(styles.DangerCardStyle.Render(
			styles.DangerStyle.Render("  Error: " + m.err),
		))
		b.WriteString("\n")
		return b.String()
	}

	if len(m.items) == 0 {
		b.WriteString(m.emptyState())
		return b.String()
	}

	// Count badge
	countBadge := styles.Badge(
		fmt.Sprintf(" %d items ", len(m.items)),
		styles.BgDark, styles.PrimaryDim,
	)
	b.WriteString("\n  " + countBadge + "\n\n")

	// Table header
	header := fmt.Sprintf("  %-3s %-30s %-12s %s", "#", "KEY", "ENV", "DESCRIPTION")
	b.WriteString(styles.MutedStyle.Render(header))
	b.WriteString("\n")
	b.WriteString("  " + styles.Divider(70))
	b.WriteString("\n")

	// Rows
	for i, env := range m.items {
		b.WriteString(m.renderRow(i, env))
		b.WriteString("\n")
	}

	return b.String()
}

func (m *EnvSecretListModel) renderRow(idx int, env models.EnvSecret) string {
	key := truncate(env.Key, 28)
	desc := truncate(env.Description, 20)
	badge := styles.EnvBadge(env.Environment)
	num := fmt.Sprintf("%-3d", idx+1)

	if idx == m.cursor {
		row := fmt.Sprintf("%-3s %-30s %s  %s", num, key, badge, desc)
		return styles.SelectedRowStyle.Render(
			styles.SelectedStyle.Render("> ") + row,
		)
	}

	return fmt.Sprintf("  %s %-30s %s  %s",
		styles.MutedStyle.Render(num),
		styles.NormalStyle.Render(key),
		badge,
		styles.MutedStyle.Render(desc),
	)
}

func (m *EnvSecretListModel) emptyState() string {
	art := lipgloss.NewStyle().Foreground(styles.Subtle).Render(`
      _____
     |     |
     | ENV |
     |_____|
      |   |
      |___|
`)

	content := lipgloss.JoinVertical(lipgloss.Center,
		art,
		styles.MutedStyle.Render("No environment secrets stored yet"),
		"",
		styles.DimStyle.Render("Press ")+styles.KeyStyle.Render("a")+styles.DimStyle.Render(" to add your first env secret"),
		styles.DimStyle.Render("Secrets can have different values per environment"),
	)
	return "\n" + content + "\n"
}
