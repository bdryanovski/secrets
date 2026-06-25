package views

import (
	"strings"

	"github.com/bdryanovski/secrets/internal/database"
	"github.com/bdryanovski/secrets/internal/models"
	"github.com/bdryanovski/secrets/internal/tui/styles"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	fieldName = iota
	fieldURL
	fieldUsername
	fieldPassword
	fieldNotes
	fieldCount
)

// CredentialFormModel handles adding/editing a credential.
type CredentialFormModel struct {
	db      *database.DB
	editing *models.Credential // nil if adding new
	inputs  []textinput.Model
	focused int
	err     string
}

// NewCredentialFormModel creates a credential form. Pass nil for a new credential.
func NewCredentialFormModel(cred *models.Credential, db *database.DB) *CredentialFormModel {
	inputs := make([]textinput.Model, fieldCount)

	for i := range inputs {
		inputs[i] = textinput.New()
		inputs[i].Width = 40
	}

	inputs[fieldName].Placeholder = "Name (e.g., GitHub)"
	inputs[fieldURL].Placeholder = "URL (e.g., https://github.com)"
	inputs[fieldUsername].Placeholder = "Username or email"
	inputs[fieldPassword].Placeholder = "Password"
	inputs[fieldPassword].EchoMode = textinput.EchoPassword
	inputs[fieldPassword].EchoCharacter = '*'
	inputs[fieldNotes].Placeholder = "Notes (optional)"

	if cred != nil {
		inputs[fieldName].SetValue(cred.Name)
		inputs[fieldURL].SetValue(cred.URL)
		inputs[fieldUsername].SetValue(cred.Username)
		inputs[fieldPassword].SetValue(cred.Password)
		inputs[fieldNotes].SetValue(cred.Notes)
	}

	inputs[fieldName].Focus()

	return &CredentialFormModel{
		db:      db,
		editing: cred,
		inputs:  inputs,
	}
}

func (m *CredentialFormModel) update(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd
	for i := range m.inputs {
		var cmd tea.Cmd
		m.inputs[i], cmd = m.inputs[i].Update(msg)
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

func (m *CredentialFormModel) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "tab", "down":
		m.inputs[m.focused].Blur()
		m.focused = (m.focused + 1) % fieldCount
		m.inputs[m.focused].Focus()
		return textinput.Blink
	case "shift+tab", "up":
		m.inputs[m.focused].Blur()
		m.focused = (m.focused - 1 + fieldCount) % fieldCount
		m.inputs[m.focused].Focus()
		return textinput.Blink
	case "enter":
		if m.focused == fieldCount-1 {
			return m.save()
		}
		// Move to next field.
		m.inputs[m.focused].Blur()
		m.focused++
		m.inputs[m.focused].Focus()
		return textinput.Blink
	case "ctrl+s":
		return m.save()
	}
	return nil
}

func (m *CredentialFormModel) save() tea.Cmd {
	name := strings.TrimSpace(m.inputs[fieldName].Value())
	if name == "" {
		m.err = "Name is required"
		return nil
	}

	cred := &models.Credential{
		Name:     name,
		URL:      strings.TrimSpace(m.inputs[fieldURL].Value()),
		Username: strings.TrimSpace(m.inputs[fieldUsername].Value()),
		Password: m.inputs[fieldPassword].Value(),
		Notes:    strings.TrimSpace(m.inputs[fieldNotes].Value()),
	}

	db := m.db
	editing := m.editing

	return func() tea.Msg {
		if editing != nil {
			cred.ID = editing.ID
			if err := db.UpdateCredential(cred); err != nil {
				return statusMsg("Error: " + err.Error())
			}
			return backToListMsg("Updated: " + cred.Name)
		}

		if err := db.CreateCredential(cred); err != nil {
			return statusMsg("Error: " + err.Error())
		}
		return backToListMsg("Added: " + cred.Name)
	}
}

func (m *CredentialFormModel) View() string {
	var b strings.Builder

	title := "Add Credential"
	if m.editing != nil {
		title = "Edit Credential"
	}
	b.WriteString("\n")
	b.WriteString(styles.TitleStyle.Render("  " + title))
	b.WriteString("\n\n")

	labels := []string{"Name:", "URL:", "Username:", "Password:", "Notes:"}
	for i, label := range labels {
		b.WriteString(styles.LabelStyle.Render("  " + label))
		b.WriteString("\n  ")
		b.WriteString(m.inputs[i].View())
		b.WriteString("\n\n")
	}

	if m.err != "" {
		b.WriteString(styles.DangerStyle.Render("  " + m.err))
		b.WriteString("\n")
	}

	b.WriteString(styles.HelpStyle.Render("  tab/shift+tab: navigate  ctrl+s/enter(last): save  esc: cancel"))
	b.WriteString("\n")

	return b.String()
}
