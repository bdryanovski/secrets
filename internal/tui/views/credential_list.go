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

// CredentialListModel displays a list of credentials.
type CredentialListModel struct {
	db     *database.DB
	items  []models.Credential
	cursor int
	err    string
}

// NewCredentialListModel creates a new credential list view.
func NewCredentialListModel(db *database.DB) *CredentialListModel {
	return &CredentialListModel{db: db}
}

type credentialsLoadedMsg struct {
	items []models.Credential
	err   error
}

func (m *CredentialListModel) loadCredentials() tea.Cmd {
	db := m.db
	return func() tea.Msg {
		items, err := db.ListCredentials()
		return credentialsLoadedMsg{items: items, err: err}
	}
}

func (m *CredentialListModel) update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case credentialsLoadedMsg:
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

func (m *CredentialListModel) handleKey(msg tea.KeyMsg) tea.Cmd {
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
	}
	return nil
}

// Selected returns the currently selected credential, or nil.
func (m *CredentialListModel) Selected() *models.Credential {
	if m.cursor >= 0 && m.cursor < len(m.items) {
		return &m.items[m.cursor]
	}
	return nil
}

func (m *CredentialListModel) View() string {
	var b strings.Builder

	if m.err != "" {
		b.WriteString("\n")
		errCard := styles.DangerCardStyle.Render(
			styles.DangerStyle.Render("  Error: " + m.err),
		)
		b.WriteString(errCard)
		b.WriteString("\n")
		return b.String()
	}

	if len(m.items) == 0 {
		b.WriteString(m.emptyState())
		return b.String()
	}

	// Item count badge
	b.WriteString("\n")
	countBadge := styles.Badge(
		fmt.Sprintf(" %d items ", len(m.items)),
		styles.BgDark, styles.PrimaryDim,
	)
	b.WriteString("  " + countBadge)
	b.WriteString("\n\n")

	// Table header
	header := fmt.Sprintf("  %-3s %-28s %-24s %s", "#", "NAME", "USERNAME", "URL")
	b.WriteString(styles.MutedStyle.Render(header))
	b.WriteString("\n")
	b.WriteString("  " + styles.Divider(70))
	b.WriteString("\n")

	// Rows
	for i, cred := range m.items {
		b.WriteString(m.renderRow(i, cred))
		b.WriteString("\n")
	}

	return b.String()
}

func (m *CredentialListModel) renderRow(idx int, cred models.Credential) string {
	name := truncate(cred.Name, 26)
	user := truncate(cred.Username, 22)
	url := truncate(cred.URL, 20)
	num := fmt.Sprintf("%-3d", idx+1)

	if idx == m.cursor {
		row := fmt.Sprintf("%-3s %-28s %-24s %s", num, name, user, url)
		return styles.SelectedRowStyle.Render(
			styles.SelectedStyle.Render("> ") + row,
		)
	}

	return fmt.Sprintf("  %s %-28s %s %s",
		styles.MutedStyle.Render(num),
		styles.NormalStyle.Render(name),
		styles.DimStyle.Render(user),
		styles.MutedStyle.Render(url),
	)
}

func (m *CredentialListModel) emptyState() string {
	art := lipgloss.NewStyle().Foreground(styles.Subtle).Render(`
       ___
      |   |
      |   |
      |___|
     /     \
    / () () \
   |   __   |
    \_______/
`)

	content := lipgloss.JoinVertical(lipgloss.Center,
		art,
		styles.MutedStyle.Render("No credentials stored yet"),
		"",
		styles.DimStyle.Render("Press ")+styles.KeyStyle.Render("a")+styles.DimStyle.Render(" to add your first credential"),
		styles.DimStyle.Render("or ")+styles.KeyStyle.Render("i")+styles.DimStyle.Render(" to import from Bitwarden / Apple"),
	)
	return "\n" + content + "\n"
}

func truncate(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}
