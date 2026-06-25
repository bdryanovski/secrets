package views

import (
	"fmt"
	"strings"

	"github.com/bdryanovski/secrets/internal/database"
	"github.com/bdryanovski/secrets/internal/models"
	"github.com/bdryanovski/secrets/internal/tui/styles"

	tea "github.com/charmbracelet/bubbletea"
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
		m.cursor = len(m.items) - 1
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
		b.WriteString(styles.DangerStyle.Render("  Error: " + m.err))
		b.WriteString("\n")
		return b.String()
	}

	if len(m.items) == 0 {
		b.WriteString(styles.MutedStyle.Render("  No credentials stored yet. Press 'a' to add one."))
		b.WriteString("\n")
		return b.String()
	}

	for i, cred := range m.items {
		cursor := "  "
		if i == m.cursor {
			cursor = styles.SelectedStyle.Render("> ")
		}

		name := cred.Name
		user := cred.Username
		if len(name) > 30 {
			name = name[:27] + "..."
		}

		line := fmt.Sprintf("%-32s %s", name, styles.MutedStyle.Render(user))

		if i == m.cursor {
			b.WriteString(cursor + styles.SelectedStyle.Render(line))
		} else {
			b.WriteString(cursor + styles.NormalStyle.Render(line))
		}
		b.WriteString("\n")
	}

	return b.String()
}
