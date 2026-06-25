package views

import (
	"fmt"
	"strings"

	"github.com/bdryanovski/secrets/internal/clipboard"
	"github.com/bdryanovski/secrets/internal/config"
	"github.com/bdryanovski/secrets/internal/password"
	"github.com/bdryanovski/secrets/internal/tui/styles"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	case "l":
		m.opts.Lowercase = !m.opts.Lowercase
		return m.generate()
	}
	return nil
}

func (m *PasswordGenModel) View() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("  " + styles.TitleStyle.Render("Password Generator"))
	b.WriteString("\n")
	b.WriteString("  " + styles.MutedStyle.Render("Generate cryptographically secure random passwords"))
	b.WriteString("\n\n")

	if m.err != "" {
		b.WriteString(styles.DangerCardStyle.Render(
			"  " + styles.DangerStyle.Render("Error: "+m.err),
		))
		b.WriteString("\n\n")
	}

	// Password display in a prominent card
	if m.password != "" {
		pwStyle := lipgloss.NewStyle().
			Foreground(styles.AccentDim).
			Bold(true)

		pwCard := styles.HighlightCardStyle.Width(58).Render(
			"\n  " + pwStyle.Render(m.password) + "\n",
		)
		b.WriteString(pwCard)
		b.WriteString("\n\n")

		// Strength meter
		b.WriteString("  " + styles.LabelStyle.Render("Strength  "))
		b.WriteString(styles.StrengthMeter(m.opts.Length, 30))
		b.WriteString("\n\n")
	}

	// Options section
	optionsCard := m.renderOptions()
	b.WriteString(styles.CardStyle.Width(58).Render(optionsCard))
	b.WriteString("\n")

	// Copy status
	if m.copied {
		b.WriteString("\n")
		b.WriteString(styles.SuccessCardStyle.Width(58).Render(
			"  " + styles.SuccessStyle.Render("Copied to clipboard!") +
				styles.MutedStyle.Render(fmt.Sprintf("  Auto-clears in %s", m.cfg.ClipboardTimeout)),
		))
	}

	return b.String()
}

func (m *PasswordGenModel) renderOptions() string {
	var b strings.Builder

	b.WriteString("  " + styles.LabelStyle.Render("Options") + "\n\n")

	// Length with visual bar
	b.WriteString("  " + styles.HeadingStyle.Render("Length: "))
	b.WriteString(styles.AccentStyle.Render(fmt.Sprintf("%d", m.opts.Length)))
	b.WriteString(styles.MutedStyle.Render("  (use + / - to adjust)"))
	b.WriteString("\n")
	b.WriteString("  " + renderLengthBar(m.opts.Length, 40))
	b.WriteString("\n\n")

	// Character set toggles
	b.WriteString("  " + styles.HeadingStyle.Render("Character Sets:") + "\n\n")

	toggles := []struct {
		label   string
		key     string
		enabled bool
		sample  string
	}{
		{"Uppercase", "u", m.opts.Uppercase, "A-Z"},
		{"Lowercase", "l", m.opts.Lowercase, "a-z"},
		{"Digits", "d", m.opts.Digits, "0-9"},
		{"Symbols", "s", m.opts.Symbols, "!@#$%..."},
	}

	for _, t := range toggles {
		b.WriteString("  " + renderToggle(t.label, t.key, t.enabled, t.sample) + "\n")
	}

	return b.String()
}

func renderToggle(label, key string, enabled bool, sample string) string {
	var toggle, text string

	if enabled {
		toggle = styles.SuccessStyle.Render("[ON] ")
		text = styles.NormalStyle.Render(label)
	} else {
		toggle = styles.MutedStyle.Render("[  ] ")
		text = styles.MutedStyle.Render(label)
	}

	keyHint := ""
	if key != "" {
		keyHint = "  " + styles.KeyStyle.Render(key)
	}

	sampleText := styles.HintStyle.Render("  " + sample)

	return toggle + text + sampleText + keyHint
}

func renderLengthBar(length, width int) string {
	pos := length - 8 // min is 8
	maxRange := 120   // 128 - 8
	filled := pos * width / maxRange
	if filled > width {
		filled = width
	}
	if filled < 1 {
		filled = 1
	}

	return lipgloss.NewStyle().Foreground(styles.Primary).Render(strings.Repeat("━", filled)) +
		lipgloss.NewStyle().Foreground(styles.Subtle).Render(strings.Repeat("─", width-filled))
}
