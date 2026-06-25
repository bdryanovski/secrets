package views

import (
	"fmt"
	"strings"

	"github.com/bdryanovski/secrets/internal/database"
	"github.com/bdryanovski/secrets/internal/importer"
	"github.com/bdryanovski/secrets/internal/tui/styles"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ImportModel handles importing from external password managers.
type ImportModel struct {
	db     *database.DB
	input  textinput.Model
	source int
	result *importer.ImportResult
	err    string
	done   bool
	saving bool
	saved  int
}

// NewImportModel creates an import view.
func NewImportModel(db *database.DB) *ImportModel {
	ti := textinput.New()
	ti.Placeholder = "/path/to/export-file.csv"
	ti.Focus()
	ti.Width = 46
	ti.PromptStyle = lipgloss.NewStyle().Foreground(styles.Primary)
	ti.TextStyle = lipgloss.NewStyle().Foreground(styles.Text)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(styles.Muted)

	return &ImportModel{
		db:    db,
		input: ti,
	}
}

type importResultMsg struct {
	result *importer.ImportResult
	err    error
}

type importSavedMsg struct {
	count int
	err   error
}

func (m *ImportModel) update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case importResultMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.result = msg.result
		}
	case importSavedMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.saved = msg.count
			m.done = true
		}
	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return cmd
	}
	return nil
}

func (m *ImportModel) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "1":
		if m.result == nil {
			m.source = 0
		}
	case "2":
		if m.result == nil {
			m.source = 1
		}
	case "enter":
		if m.result != nil && !m.saving {
			return m.saveImported()
		}
		return m.runImport()
	}
	return nil
}

func (m *ImportModel) runImport() tea.Cmd {
	filePath := strings.TrimSpace(m.input.Value())
	if filePath == "" {
		m.err = "File path is required"
		return nil
	}

	source := m.source
	return func() tea.Msg {
		var imp importer.Importer
		if source == 0 {
			imp = &importer.BitwardenImporter{}
		} else {
			imp = &importer.AppleImporter{}
		}
		result, err := imp.Import(filePath)
		return importResultMsg{result: result, err: err}
	}
}

func (m *ImportModel) saveImported() tea.Cmd {
	m.saving = true
	db := m.db
	creds := m.result.Credentials

	return func() tea.Msg {
		count := 0
		for _, c := range creds {
			cred := c
			if err := db.CreateCredential(&cred); err != nil {
				return importSavedMsg{count: count, err: err}
			}
			count++
		}
		return importSavedMsg{count: count}
	}
}

func (m *ImportModel) View() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("  " + styles.TitleStyle.Render("Import Credentials"))
	b.WriteString("\n")
	b.WriteString("  " + styles.MutedStyle.Render("Import credentials from external password managers"))
	b.WriteString("\n\n")

	if m.done {
		b.WriteString(m.viewDone())
		return b.String()
	}

	// Step indicator
	step := 1
	if m.result != nil {
		step = 3
	}
	b.WriteString("  " + styles.ProgressDots(step-1, 3))
	b.WriteString(styles.MutedStyle.Render("  Step " + fmt.Sprintf("%d", step) + " of 3"))
	b.WriteString("\n\n")

	// Step 1: Source selection
	b.WriteString(styles.LabelStyle.Render("  Source"))
	b.WriteString("\n")
	b.WriteString(styles.MutedStyle.Render("  Choose which password manager you exported from"))
	b.WriteString("\n\n")

	sources := []struct {
		name, desc, formats string
	}{
		{"Bitwarden", "Open-source password manager", "CSV, JSON"},
		{"Apple Passwords", "macOS / iOS built-in keychain", "CSV"},
	}
	for i, src := range sources {
		if i == m.source {
			card := styles.HighlightCardStyle.Width(52).Render(
				styles.SelectedStyle.Render(" > "+src.name) + "\n" +
					"   " + styles.MutedStyle.Render(src.desc) + "\n" +
					"   " + styles.HintStyle.Render("Formats: "+src.formats),
			)
			b.WriteString(card)
		} else {
			card := styles.CardStyle.Width(52).Render(
				styles.DimStyle.Render("   "+src.name) + "\n" +
					"   " + styles.MutedStyle.Render(src.desc),
			)
			b.WriteString(card)
		}
		b.WriteString("\n")
	}

	// Step 2: File path
	b.WriteString("\n")
	b.WriteString(styles.LabelStyle.Render("  File Path"))
	b.WriteString("\n")
	b.WriteString(styles.MutedStyle.Render("  Path to the exported file on your filesystem"))
	b.WriteString("\n\n")
	b.WriteString(styles.InputGroupFocusedStyle.Width(54).Render("  " + m.input.View()))
	b.WriteString("\n")

	if m.err != "" {
		b.WriteString("\n")
		b.WriteString("  " + styles.DangerStyle.Render("! "+m.err))
		b.WriteString("\n")
	}

	// Step 3: Review (if parsed)
	if m.result != nil {
		b.WriteString("\n")
		b.WriteString(m.viewReview())
	}

	return b.String()
}

func (m *ImportModel) viewReview() string {
	var b strings.Builder

	b.WriteString(styles.LabelStyle.Render("  Review"))
	b.WriteString("\n\n")

	stats := fmt.Sprintf("  Total: %d   Ready: %d   Skipped: %d",
		m.result.Total,
		len(m.result.Credentials),
		m.result.Skipped,
	)
	b.WriteString(styles.SuccessCardStyle.Width(54).Render(
		styles.SuccessStyle.Render("  Parsed successfully!") + "\n" +
			styles.NormalStyle.Render(stats) + "\n\n" +
			"  Press " + styles.KeyStyle.Render("enter") + " to import all credentials",
	))
	b.WriteString("\n")

	return b.String()
}

func (m *ImportModel) viewDone() string {
	content := lipgloss.JoinVertical(lipgloss.Center,
		"",
		styles.SuccessStyle.Render(fmt.Sprintf("  Successfully imported %d credentials!", m.saved)),
		"",
		styles.MutedStyle.Render("  Your imported credentials are now encrypted and stored"),
		styles.MutedStyle.Render("  in your local vault."),
		"",
	)
	return styles.SuccessCardStyle.Width(56).Render(content)
}
