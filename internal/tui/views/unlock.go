package views

import (
	"fmt"
	"os"
	"strings"

	"github.com/bdryanovski/secrets/internal/config"
	"github.com/bdryanovski/secrets/internal/database"
	"github.com/bdryanovski/secrets/internal/tui/styles"
	"github.com/bdryanovski/secrets/internal/version"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// UnlockModel is the master password entry screen.
type UnlockModel struct {
	cfg       *config.Config
	input     textinput.Model
	err       string
	isNew     bool // true if this is a new database (first run)
	confirm   textinput.Model
	stage     int // 0 = enter password, 1 = confirm password (new DB only)
	firstPass string
}

// NewUnlockModel creates the unlock screen.
func NewUnlockModel(cfg *config.Config) *UnlockModel {
	ti := textinput.New()
	ti.Placeholder = "Enter master password"
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '*'
	ti.Focus()
	ti.Width = 40

	ci := textinput.New()
	ci.Placeholder = "Confirm master password"
	ci.EchoMode = textinput.EchoPassword
	ci.EchoCharacter = '*'
	ci.Width = 40

	// Check if database already exists.
	isNew := !fileExists(cfg.DBPath)

	return &UnlockModel{
		cfg:     cfg,
		input:   ti,
		confirm: ci,
		isNew:   isNew,
		stage:   0,
	}
}

// dbOpenedMsg is sent when the database is successfully opened.
type dbOpenedMsg struct {
	db *database.DB
}

// errMsg is sent when an error occurs.
type errMsg struct {
	err string
}

func (m *UnlockModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *UnlockModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			return m.handleSubmit()
		}

	case dbOpenedMsg:
		return NewMainModel(m.cfg, msg.db), nil

	case errMsg:
		m.err = msg.err
		m.input.SetValue("")
		m.input.Focus()
		return m, nil
	}

	var cmd tea.Cmd
	if m.stage == 0 {
		m.input, cmd = m.input.Update(msg)
	} else {
		m.confirm, cmd = m.confirm.Update(msg)
	}
	return m, cmd
}

func (m *UnlockModel) handleSubmit() (tea.Model, tea.Cmd) {
	if m.isNew {
		// New database: need password + confirmation.
		if m.stage == 0 {
			pw := m.input.Value()
			if len(pw) < 8 {
				m.err = "Password must be at least 8 characters"
				return m, nil
			}
			m.firstPass = pw
			m.stage = 1
			m.err = ""
			m.input.Blur()
			m.confirm.Focus()
			return m, textinput.Blink
		}

		// Stage 1: confirm password.
		if m.confirm.Value() != m.firstPass {
			m.err = "Passwords do not match"
			m.confirm.SetValue("")
			return m, nil
		}
	}

	password := m.input.Value()
	if m.isNew {
		password = m.firstPass
	}

	cfg := m.cfg
	return m, func() tea.Msg {
		if err := cfg.EnsureConfigDir(); err != nil {
			return errMsg{err: fmt.Sprintf("Failed to create config dir: %s", err)}
		}

		db, err := database.Open(cfg.DBPath, password)
		if err != nil {
			return errMsg{err: fmt.Sprintf("Failed to unlock: %s", err)}
		}
		return dbOpenedMsg{db: db}
	}
}

func (m *UnlockModel) View() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(styles.TitleStyle.Render("  Secrets Manager"))
	b.WriteString("\n")
	b.WriteString(styles.MutedStyle.Render("  " + version.String()))
	b.WriteString("\n\n")

	if m.isNew {
		b.WriteString(styles.SuccessStyle.Render("  Creating new vault"))
		b.WriteString("\n\n")
		if m.stage == 0 {
			b.WriteString(styles.LabelStyle.Render("  Master Password:"))
			b.WriteString("\n  ")
			b.WriteString(m.input.View())
		} else {
			b.WriteString(styles.LabelStyle.Render("  Confirm Password:"))
			b.WriteString("\n  ")
			b.WriteString(m.confirm.View())
		}
	} else {
		b.WriteString(styles.LabelStyle.Render("  Master Password:"))
		b.WriteString("\n  ")
		b.WriteString(m.input.View())
	}

	if m.err != "" {
		b.WriteString("\n\n")
		b.WriteString(styles.DangerStyle.Render("  " + m.err))
	}

	b.WriteString("\n\n")
	b.WriteString(styles.HelpStyle.Render("  enter: submit  •  esc: quit"))
	b.WriteString("\n")

	return b.String()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
