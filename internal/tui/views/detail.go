package views

import (
	"fmt"
	"strings"

	"github.com/bdryanovski/secrets/internal/clipboard"
	"github.com/bdryanovski/secrets/internal/config"
	"github.com/bdryanovski/secrets/internal/database"
	"github.com/bdryanovski/secrets/internal/models"
	"github.com/bdryanovski/secrets/internal/tui/styles"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DetailType identifies what kind of item is being shown.
type DetailType int

const (
	DetailCredential DetailType = iota
	DetailEnvSecret
)

// DetailModel shows the full details of a credential or env secret.
type DetailModel struct {
	db         *database.DB
	cfg        *config.Config
	itemID     int64
	detailType DetailType

	name     string
	url      string
	username string
	password string
	notes    string
	env      string
	key      string
	value    string

	showPassword bool
	copied       bool
	err          string
}

// NewDetailModel creates a detail view for the given item.
func NewDetailModel(db *database.DB, cfg *config.Config, id int64, dt DetailType) *DetailModel {
	return &DetailModel{
		db:         db,
		cfg:        cfg,
		itemID:     id,
		detailType: dt,
	}
}

type detailLoadedMsg struct {
	err error
}

func (m *DetailModel) load() tea.Cmd {
	db := m.db
	id := m.itemID
	dt := m.detailType
	return func() tea.Msg {
		if dt == DetailCredential {
			cred, err := db.GetCredential(id)
			if err != nil {
				return detailLoadedMsg{err: err}
			}
			return credDetailMsg{cred: cred}
		}
		env, err := db.GetEnvSecret(id)
		if err != nil {
			return detailLoadedMsg{err: err}
		}
		return envDetailMsg{env: env}
	}
}

type credDetailMsg struct {
	cred *models.Credential
}

type envDetailMsg struct {
	env *models.EnvSecret
}

func (m *DetailModel) update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case detailLoadedMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
		}
	case credDetailMsg:
		m.name = msg.cred.Name
		m.url = msg.cred.URL
		m.username = msg.cred.Username
		m.password = msg.cred.Password
		m.notes = msg.cred.Notes
	case envDetailMsg:
		m.key = msg.env.Key
		m.value = msg.env.Value
		m.env = msg.env.Environment
		m.notes = msg.env.Description
	}
	return nil
}

func (m *DetailModel) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "s":
		m.showPassword = !m.showPassword
	case "c":
		secret := m.password
		if m.detailType == DetailEnvSecret {
			secret = m.value
		}
		if secret != "" {
			_, err := clipboard.CopyWithTimeout(secret, m.cfg.ClipboardTimeout)
			if err != nil {
				m.err = err.Error()
			} else {
				m.copied = true
			}
		}
	}
	return nil
}

func (m *DetailModel) View() string {
	if m.err != "" {
		return "\n" + styles.DangerCardStyle.Render(
			styles.DangerStyle.Render("  Error: "+m.err),
		)
	}

	var b strings.Builder
	b.WriteString("\n")

	if m.detailType == DetailCredential {
		b.WriteString(m.renderCredentialCard())
	} else {
		b.WriteString(m.renderEnvSecretCard())
	}

	if m.copied {
		b.WriteString("\n")
		b.WriteString(styles.SuccessCardStyle.Width(56).Render(
			"  " + styles.SuccessStyle.Render("Copied to clipboard!") + "\n" +
				"  " + styles.MutedStyle.Render(fmt.Sprintf("Will auto-clear in %s", m.cfg.ClipboardTimeout)),
		))
	}

	return b.String()
}

func (m *DetailModel) renderCredentialCard() string {
	visibilityIcon := "  [hidden]"
	if m.showPassword {
		visibilityIcon = "  [visible]"
	}

	pw := strings.Repeat("● ", 8)
	if m.showPassword {
		pw = m.password
	}

	rows := []string{
		m.detailRow("Name", m.name, styles.Text),
		m.detailRow("URL", m.url, styles.Accent),
		m.detailRow("Username", m.username, styles.Text),
	}

	// Password row with visibility indicator
	pwLabel := styles.LabelStyle.Render("  Password    ")
	pwValue := lipgloss.NewStyle().Foreground(styles.WarningDim).Render(pw)
	pwHint := styles.MutedStyle.Render(visibilityIcon)
	rows = append(rows, pwLabel+pwValue+pwHint)

	if m.notes != "" {
		rows = append(rows, "")
		rows = append(rows, m.detailRow("Notes", m.notes, styles.TextDim))
	}

	header := "  " + styles.TitleStyle.Render("Credential Details")
	content := lipgloss.JoinVertical(lipgloss.Left,
		append([]string{header, ""}, rows...)...,
	)

	return styles.HighlightCardStyle.Width(56).Render(content)
}

func (m *DetailModel) renderEnvSecretCard() string {
	visibilityIcon := "  [hidden]"
	if m.showPassword {
		visibilityIcon = "  [visible]"
	}

	val := strings.Repeat("● ", 8)
	if m.showPassword {
		val = m.value
	}

	rows := []string{
		m.detailRow("Key", m.key, styles.Text),
		"  " + styles.LabelStyle.Render("Environment   ") + styles.EnvBadge(m.env),
	}

	// Value row
	valLabel := styles.LabelStyle.Render("  Value       ")
	valValue := lipgloss.NewStyle().Foreground(styles.WarningDim).Render(val)
	valHint := styles.MutedStyle.Render(visibilityIcon)
	rows = append(rows, valLabel+valValue+valHint)

	if m.notes != "" {
		rows = append(rows, "")
		rows = append(rows, m.detailRow("Description", m.notes, styles.TextDim))
	}

	// Shell usage hint
	rows = append(rows, "")
	rows = append(rows,
		styles.MutedStyle.Render("  Shell usage:"),
	)
	rows = append(rows,
		styles.AccentStyle.Render("  eval $(secrets env -p "+m.env+")"),
	)

	header := "  " + styles.TitleStyle.Render("Env Secret Details")
	content := lipgloss.JoinVertical(lipgloss.Left,
		append([]string{header, ""}, rows...)...,
	)

	return styles.HighlightCardStyle.Width(56).Render(content)
}

func (m *DetailModel) detailRow(label, value string, color lipgloss.Color) string {
	padded := fmt.Sprintf("%-14s", label)
	return "  " + styles.LabelStyle.Render(padded) +
		lipgloss.NewStyle().Foreground(color).Render(value)
}
