package views

import (
	"fmt"
	"strings"

	"github.com/bdryanovski/secrets/internal/database"
	"github.com/bdryanovski/secrets/internal/models"
	"github.com/bdryanovski/secrets/internal/tui/styles"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	ti.Width = 48
	ti.PromptStyle = lipgloss.NewStyle().Foreground(styles.Primary)
	ti.TextStyle = lipgloss.NewStyle().Foreground(styles.Text)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(styles.Muted)

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
	b.WriteString("  " + styles.TitleStyle.Render("Search"))
	b.WriteString("\n")
	b.WriteString("  " + styles.MutedStyle.Render("Find credentials and env secrets by name, key, or URL"))
	b.WriteString("\n\n")

	// Search input in a card
	searchBox := styles.InputGroupFocusedStyle.Width(54).Render(
		"  " + m.input.View(),
	)
	b.WriteString(searchBox)
	b.WriteString("\n")

	if m.err != "" {
		b.WriteString("\n")
		b.WriteString(styles.DangerCardStyle.Render(
			"  " + styles.DangerStyle.Render("Error: "+m.err),
		))
		b.WriteString("\n")
		return b.String()
	}

	if !m.queried {
		b.WriteString("\n")
		b.WriteString("  " + styles.MutedStyle.Render("Press Enter to search"))
		b.WriteString("\n")
		return b.String()
	}

	total := len(m.creds) + len(m.envs)
	b.WriteString("\n")
	if total == 0 {
		b.WriteString("  " + styles.MutedStyle.Render("No results found"))
		b.WriteString("\n")
		return b.String()
	}

	resultBadge := styles.Badge(
		fmt.Sprintf(" %d results ", total),
		styles.BgDark, styles.PrimaryDim,
	)
	b.WriteString("  " + resultBadge + "\n\n")

	if len(m.creds) > 0 {
		credHeader := styles.Badge(" Credentials ", styles.BgDark, styles.Accent)
		b.WriteString("  " + credHeader + "\n\n")

		for _, c := range m.creds {
			name := truncate(c.Name, 24)
			user := truncate(c.Username, 20)
			url := truncate(c.URL, 24)
			b.WriteString(fmt.Sprintf("    %s  %s  %s\n",
				styles.NormalStyle.Render(name),
				styles.DimStyle.Render(user),
				styles.MutedStyle.Render(url),
			))
		}
		b.WriteString("\n")
	}

	if len(m.envs) > 0 {
		envHeader := styles.Badge(" Env Secrets ", styles.BgDark, styles.Success)
		b.WriteString("  " + envHeader + "\n\n")

		for _, e := range m.envs {
			key := truncate(e.Key, 28)
			b.WriteString(fmt.Sprintf("    %s  %s\n",
				styles.NormalStyle.Render(key),
				styles.EnvBadge(e.Environment),
			))
		}
	}

	return b.String()
}
