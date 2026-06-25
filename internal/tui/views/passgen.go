package views

import (
	"fmt"
	"strings"

	"github.com/bdryanovski/secrets/internal/clipboard"
	"github.com/bdryanovski/secrets/internal/config"
	"github.com/bdryanovski/secrets/internal/password"
	"github.com/bdryanovski/secrets/internal/tui/styles"

	tea "github.com/charmbracelet/bubbletea"
)

// PasswordGenModel handles generating random passwords.
type PasswordGenModel struct {
	cfg      *config.Config
	password string
	opts     password.Options
	err      string
	copied   bool
}

// NewPasswordGenModel creates a password generator view.
func NewPasswordGenModel(cfg *config.Config) *PasswordGenModel {
	opts := password.DefaultOptions()
	opts.Length = cfg.PasswordLength

	return &PasswordGenModel{
		cfg:  cfg,
		opts: opts,
	}
}

type passwordGeneratedMsg struct {
	password string
	err      error
}

func (m *PasswordGenModel) Init() tea.Cmd {
	return m.generate()
}

func (m *PasswordGenModel) generate() tea.Cmd {
	opts := m.opts
	return func() tea.Msg {
		pw, err := password.Generate(opts)
		return passwordGeneratedMsg{password: pw, err: err}
	}
}

func (m *PasswordGenModel) update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case passwordGeneratedMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.password = msg.password
			m.err = ""
			m.copied = false
		}
	}
	return nil
}

func (m *PasswordGenModel) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "r":
		return m.generate()
	case "c":
		if m.password != "" {
			_, err := clipboard.CopyWithTimeout(m.password, m.cfg.ClipboardTimeout)
			if err != nil {
				m.err = err.Error()
			} else {
				m.copied = true
			}
		}
	case "+":
		if m.opts.Length < 128 {
			m.opts.Length++
			return m.generate()
		}
	case "-":
		if m.opts.Length > 8 {
			m.opts.Length--
			return m.generate()
		}
	case "s":
		m.opts.Symbols = !m.opts.Symbols
		return m.generate()
	case "d":
		m.opts.Digits = !m.opts.Digits
		return m.generate()
	case "u":
		m.opts.Uppercase = !m.opts.Uppercase
		return m.generate()
	}
	return nil
}

func (m *PasswordGenModel) View() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(styles.TitleStyle.Render("  Password Generator"))
	b.WriteString("\n\n")

	if m.err != "" {
		b.WriteString(styles.DangerStyle.Render("  Error: " + m.err))
		b.WriteString("\n\n")
	}

	if m.password != "" {
		b.WriteString(styles.BoxStyle.Render(m.password))
		b.WriteString("\n\n")
	}

	// Options display
	b.WriteString(styles.LabelStyle.Render(fmt.Sprintf("  Length: %d", m.opts.Length)))
	b.WriteString(styles.MutedStyle.Render("  (+/- to adjust)"))
	b.WriteString("\n")

	b.WriteString("  ")
	b.WriteString(toggleLabel("Uppercase (u)", m.opts.Uppercase))
	b.WriteString("  ")
	b.WriteString(toggleLabel("Lowercase", m.opts.Lowercase))
	b.WriteString("  ")
	b.WriteString(toggleLabel("Digits (d)", m.opts.Digits))
	b.WriteString("  ")
	b.WriteString(toggleLabel("Symbols (s)", m.opts.Symbols))
	b.WriteString("\n\n")

	if m.copied {
		b.WriteString(styles.SuccessStyle.Render(fmt.Sprintf("  Copied! Will clear in %s", m.cfg.ClipboardTimeout)))
		b.WriteString("\n\n")
	}

	b.WriteString(styles.HelpStyle.Render("  r: regenerate  c: copy  +/-: length  u/d/s: toggle  esc: back"))
	b.WriteString("\n")

	return b.String()
}

func toggleLabel(label string, enabled bool) string {
	if enabled {
		return styles.SuccessStyle.Render("[x] " + label)
	}
	return styles.MutedStyle.Render("[ ] " + label)
}
