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
		// Copy password/value to clipboard with timeout.
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
	var b strings.Builder

	if m.err != "" {
		b.WriteString(styles.DangerStyle.Render("  Error: " + m.err))
		b.WriteString("\n")
		return b.String()
	}

	if m.detailType == DetailCredential {
		b.WriteString(m.renderCredential())
	} else {
		b.WriteString(m.renderEnvSecret())
	}

	// Help line
	b.WriteString("\n")
	b.WriteString(styles.HelpStyle.Render("  s: show/hide password  c: copy  e: edit  esc: back"))
	b.WriteString("\n")

	if m.copied {
		b.WriteString(styles.SuccessStyle.Render(fmt.Sprintf("  Copied to clipboard! Will clear in %s", m.cfg.ClipboardTimeout)))
		b.WriteString("\n")
	}

	return b.String()
}

func (m *DetailModel) renderCredential() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(styles.LabelStyle.Render("  Name:     ") + m.name + "\n")
	b.WriteString(styles.LabelStyle.Render("  URL:      ") + m.url + "\n")
	b.WriteString(styles.LabelStyle.Render("  Username: ") + m.username + "\n")

	pw := strings.Repeat("*", 12)
	if m.showPassword {
		pw = m.password
	}
	b.WriteString(styles.LabelStyle.Render("  Password: ") + pw + "\n")

	if m.notes != "" {
		b.WriteString(styles.LabelStyle.Render("  Notes:    ") + m.notes + "\n")
	}
	return b.String()
}

func (m *DetailModel) renderEnvSecret() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(styles.LabelStyle.Render("  Key:         ") + m.key + "\n")
	b.WriteString(styles.LabelStyle.Render("  Environment: ") + envBadge(m.env) + "\n")

	val := strings.Repeat("*", 12)
	if m.showPassword {
		val = m.value
	}
	b.WriteString(styles.LabelStyle.Render("  Value:       ") + val + "\n")

	if m.notes != "" {
		b.WriteString(styles.LabelStyle.Render("  Description: ") + m.notes + "\n")
	}
	return b.String()
}
