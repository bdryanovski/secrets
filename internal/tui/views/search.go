package views

import (
	"fmt"
	"strings"

	"github.com/bdryanovski/secrets/internal/database"
	"github.com/bdryanovski/secrets/internal/models"
	"github.com/bdryanovski/secrets/internal/tui/styles"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// SearchModel handles searching across credentials and env secrets.
type SearchModel struct {
	db      *database.DB
	input   textinput.Model
	creds   []models.Credential
	envs    []models.EnvSecret
	err     string
	queried bool
}

// NewSearchModel creates a search view.
func NewSearchModel(db *database.DB) *SearchModel {
	ti := textinput.New()
	ti.Placeholder = "Search credentials and env secrets..."
	ti.Focus()
	ti.Width = 50

	return &SearchModel{
		db:    db,
		input: ti,
	}
}

type searchResultsMsg struct {
	creds []models.Credential
	envs  []models.EnvSecret
	err   error
}

func (m *SearchModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *SearchModel) update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case searchResultsMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.creds = msg.creds
			m.envs = msg.envs
			m.queried = true
		}
	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return cmd
	}
	return nil
}

func (m *SearchModel) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		query := strings.TrimSpace(m.input.Value())
		if query == "" {
			return nil
		}
		db := m.db
		return func() tea.Msg {
			creds, err1 := db.SearchCredentials(query)
			envs, err2 := db.SearchEnvSecrets(query)
			if err1 != nil {
				return searchResultsMsg{err: err1}
			}
			if err2 != nil {
				return searchResultsMsg{err: err2}
			}
			return searchResultsMsg{creds: creds, envs: envs}
		}
	}
	return nil
}

func (m *SearchModel) View() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(styles.TitleStyle.Render("  Search"))
	b.WriteString("\n\n")
	b.WriteString("  ")
	b.WriteString(m.input.View())
	b.WriteString("\n\n")

	if m.err != "" {
		b.WriteString(styles.DangerStyle.Render("  Error: " + m.err))
		b.WriteString("\n")
		return b.String()
	}

	if !m.queried {
		b.WriteString(styles.MutedStyle.Render("  Type a query and press Enter"))
		b.WriteString("\n")
		return b.String()
	}

	total := len(m.creds) + len(m.envs)
	b.WriteString(styles.MutedStyle.Render(fmt.Sprintf("  Found %d results", total)))
	b.WriteString("\n\n")

	if len(m.creds) > 0 {
		b.WriteString(styles.LabelStyle.Render("  Credentials:"))
		b.WriteString("\n")
		for _, c := range m.creds {
			b.WriteString(fmt.Sprintf("    %s  %s  %s\n",
				styles.NormalStyle.Render(c.Name),
				styles.MutedStyle.Render(c.Username),
				styles.MutedStyle.Render(c.URL),
			))
		}
		b.WriteString("\n")
	}

	if len(m.envs) > 0 {
		b.WriteString(styles.LabelStyle.Render("  Env Secrets:"))
		b.WriteString("\n")
		for _, e := range m.envs {
			b.WriteString(fmt.Sprintf("    %s  %s\n",
				styles.NormalStyle.Render(e.Key),
				envBadge(e.Environment),
			))
		}
	}

	b.WriteString("\n")
	b.WriteString(styles.HelpStyle.Render("  enter: search  esc: back"))
	b.WriteString("\n")

	return b.String()
}
