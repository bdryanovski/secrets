package views

import (
	"fmt"
	"os"
	"strings"

	"github.com/bdryanovski/secrets/internal/config"
	"github.com/bdryanovski/secrets/internal/database"
	"github.com/bdryanovski/secrets/internal/tui/styles"
	"github.com/bdryanovski/secrets/internal/version"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// unlockPhase tracks which screen we're on.
type unlockPhase int

const (
	phaseSplash unlockPhase = iota // Show logo briefly
	phaseInput                     // Enter master password
	phaseUnlock                    // Spinner while decrypting
)

// UnlockModel is the master password entry screen.
type UnlockModel struct {
	cfg       *config.Config
	input     textinput.Model
	confirm   textinput.Model
	spinner   spinner.Model
	err       string
	isNew     bool
	stage     int // 0 = enter password, 1 = confirm (new DB only)
	firstPass string
	phase     unlockPhase
	width     int
	height    int
}

// NewUnlockModel creates the unlock screen.
func NewUnlockModel(cfg *config.Config) *UnlockModel {
	ti := textinput.New()
	ti.Placeholder = "Enter master password..."
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '●'
	ti.Focus()
	ti.Width = 44
	ti.PromptStyle = lipgloss.NewStyle().Foreground(styles.Primary)
	ti.TextStyle = lipgloss.NewStyle().Foreground(styles.Text)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(styles.Muted)

	ci := textinput.New()
	ci.Placeholder = "Confirm master password..."
	ci.EchoMode = textinput.EchoPassword
	ci.EchoCharacter = '●'
	ci.Width = 44
	ci.PromptStyle = lipgloss.NewStyle().Foreground(styles.Primary)
	ci.TextStyle = lipgloss.NewStyle().Foreground(styles.Text)
	ci.PlaceholderStyle = lipgloss.NewStyle().Foreground(styles.Muted)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = styles.SpinnerStyle

	isNew := !fileExists(cfg.DBPath)

	return &UnlockModel{
		cfg:     cfg,
		input:   ti,
		confirm: ci,
		spinner: sp,
		isNew:   isNew,
		phase:   phaseSplash,
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
	return m.spinner.Tick
}

func (m *UnlockModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.phase == phaseSplash {
			// Any key skips the splash.
			m.phase = phaseInput
			return m, textinput.Blink
		}
		if m.phase == phaseUnlock {
			return m, nil // Ignore keys while unlocking.
		}
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			return m.handleSubmit()
		}

	case dbOpenedMsg:
		main := NewMainModel(m.cfg, msg.db, m.width, m.height)
		return main, main.Init()

	case errMsg:
		m.phase = phaseInput
		m.err = msg.err
		m.input.SetValue("")
		m.input.Focus()
		return m, textinput.Blink

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	// Forward to text inputs.
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

	m.phase = phaseUnlock
	m.err = ""
	cfg := m.cfg

	return m, tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			if err := cfg.EnsureConfigDir(); err != nil {
				return errMsg{err: fmt.Sprintf("Failed to create config dir: %s", err)}
			}
			db, err := database.Open(cfg.DBPath, password)
			if err != nil {
				return errMsg{err: fmt.Sprintf("Failed to unlock: %s", err)}
			}
			return dbOpenedMsg{db: db}
		},
	)
}

func (m *UnlockModel) View() string {
	switch m.phase {
	case phaseSplash:
		return m.viewSplash()
	case phaseUnlock:
		return m.viewUnlocking()
	default:
		return m.viewInput()
	}
}

func (m *UnlockModel) viewSplash() string {
	content := lipgloss.JoinVertical(lipgloss.Center,
		"",
		styles.Logo(),
		"",
		styles.MutedStyle.Render("Your secrets, locally encrypted"),
		"",
		styles.DimStyle.Render("v"+version.Version),
		"",
		styles.MutedStyle.Render("Press any key to continue..."),
	)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m *UnlockModel) viewUnlocking() string {
	card := styles.HighlightCardStyle.Width(54).Render(
		lipgloss.JoinVertical(lipgloss.Center,
			"",
			m.spinner.View()+"  "+styles.AccentStyle.Render("Unlocking vault..."),
			"",
			styles.MutedStyle.Render("Deriving encryption key with Argon2id"),
			"",
		),
	)

	content := lipgloss.JoinVertical(lipgloss.Center,
		"",
		styles.LogoSmall(),
		"",
		card,
	)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m *UnlockModel) viewInput() string {
	var cardContent strings.Builder

	if m.isNew {
		cardContent.WriteString(styles.SuccessStyle.Render("  Creating new vault") + "\n")
		cardContent.WriteString(styles.MutedStyle.Render("  Choose a strong master password. This encrypts") + "\n")
		cardContent.WriteString(styles.MutedStyle.Render("  your entire database and cannot be recovered.") + "\n")
		cardContent.WriteString("\n")

		if m.stage == 0 {
			cardContent.WriteString(styles.LabelStyle.Render("  Master Password") + "\n")
			cardContent.WriteString(styles.HintStyle.Render("  Minimum 8 characters") + "\n\n")
			cardContent.WriteString("  " + m.input.View() + "\n")
			cardContent.WriteString("\n")
			cardContent.WriteString("  " + styles.ProgressDots(0, 2) + styles.MutedStyle.Render("  Step 1 of 2"))
		} else {
			cardContent.WriteString(styles.LabelStyle.Render("  Confirm Password") + "\n")
			cardContent.WriteString(styles.HintStyle.Render("  Re-enter to verify") + "\n\n")
			cardContent.WriteString("  " + m.confirm.View() + "\n")
			cardContent.WriteString("\n")
			cardContent.WriteString("  " + styles.ProgressDots(1, 2) + styles.MutedStyle.Render("  Step 2 of 2"))
		}
	} else {
		cardContent.WriteString(styles.AccentStyle.Render("  Unlock your vault") + "\n")
		cardContent.WriteString(styles.MutedStyle.Render("  Enter your master password to decrypt") + "\n")
		cardContent.WriteString(styles.MutedStyle.Render("  your secrets database.") + "\n")
		cardContent.WriteString("\n")
		cardContent.WriteString(styles.LabelStyle.Render("  Master Password") + "\n\n")
		cardContent.WriteString("  " + m.input.View())
	}

	if m.err != "" {
		cardContent.WriteString("\n\n")
		cardContent.WriteString("  " + styles.DangerStyle.Render("! "+m.err))
	}

	card := styles.HighlightCardStyle.Width(54).Render(cardContent.String())

	help := styles.HelpBar(
		styles.KeyHint("enter", "submit"),
		styles.KeyHint("esc", "quit"),
	)

	content := lipgloss.JoinVertical(lipgloss.Center,
		"",
		styles.LogoSmall(),
		styles.DimStyle.Render("v"+version.Version),
		"",
		card,
		"",
		help,
	)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
