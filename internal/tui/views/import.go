package views

import (
	"fmt"
	"strings"

	"github.com/bdryanovski/secrets/internal/database"
	"github.com/bdryanovski/secrets/internal/importer"
	"github.com/bdryanovski/secrets/internal/tui/styles"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// ImportModel handles importing from external password managers.
type ImportModel struct {
	db     *database.DB
	input  textinput.Model
	source int // 0 = bitwarden, 1 = apple
	result *importer.ImportResult
	err    string
	done   bool
	saving bool
	saved  int
}

// NewImportModel creates an import view.
func NewImportModel(db *database.DB) *ImportModel {
	ti := textinput.New()
	ti.Placeholder = "Path to export file..."
	ti.Focus()
	ti.Width = 50

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
			cred := c // copy for closure
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
	b.WriteString(styles.TitleStyle.Render("  Import"))
	b.WriteString("\n\n")

	if m.done {
		b.WriteString(styles.SuccessStyle.Render(fmt.Sprintf("  Successfully imported %d credentials!", m.saved)))
		b.WriteString("\n\n")
		b.WriteString(styles.HelpStyle.Render("  esc: back"))
		return b.String()
	}

	// Source selector
	b.WriteString(styles.LabelStyle.Render("  Source:"))
	b.WriteString("\n")
	sources := []string{"Bitwarden (1)", "Apple Passwords (2)"}
	for i, s := range sources {
		if i == m.source {
			b.WriteString("  " + styles.SelectedStyle.Render("> "+s))
		} else {
			b.WriteString("  " + styles.MutedStyle.Render("  "+s))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")

	b.WriteString(styles.LabelStyle.Render("  File path:"))
	b.WriteString("\n  ")
	b.WriteString(m.input.View())
	b.WriteString("\n\n")

	if m.err != "" {
		b.WriteString(styles.DangerStyle.Render("  " + m.err))
		b.WriteString("\n\n")
	}

	if m.result != nil {
		b.WriteString(styles.SuccessStyle.Render(fmt.Sprintf("  Parsed: %d items (%d skipped)", m.result.Total, m.result.Skipped)))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  Ready to import %d credentials", len(m.result.Credentials)))
		b.WriteString("\n\n")
		b.WriteString(styles.HelpStyle.Render("  enter: save all  esc: cancel"))
	} else {
		b.WriteString(styles.HelpStyle.Render("  1/2: select source  enter: import  esc: back"))
	}
	b.WriteString("\n")

	return b.String()
}
