package views

import (
	"strings"

	"github.com/bdryanovski/secrets/internal/database"
	"github.com/bdryanovski/secrets/internal/models"
	"github.com/bdryanovski/secrets/internal/tui/styles"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	fieldName = iota
	fieldURL
	fieldUsername
	fieldPassword
	fieldNotes
	fieldCount
)

// fieldMeta holds label, hint, and description for each form field.
type fieldMeta struct {
	label string
	hint  string
	desc  string
}

var credentialFields = []fieldMeta{
	{
		label: "Name",
		hint:  "Required",
		desc:  "A friendly name for this credential (e.g., \"GitHub\", \"Work Email\")",
	},
	{
		label: "URL",
		hint:  "Optional",
		desc:  "The website login page URL for quick reference",
	},
	{
		label: "Username",
		hint:  "Optional",
		desc:  "Your login username or email address for this service",
	},
	{
		label: "Password",
		hint:  "Will be encrypted",
		desc:  "The password is encrypted with AES-256-GCM before storage",
	},
	{
		label: "Notes",
		hint:  "Optional",
		desc:  "Any additional information, recovery codes, or reminders",
	},
}

// CredentialFormModel handles adding/editing a credential.
type CredentialFormModel struct {
	db      *database.DB
	editing *models.Credential
	inputs  []textinput.Model
	focused int
	err     string
}

// NewCredentialFormModel creates a credential form. Pass nil for a new credential.
func NewCredentialFormModel(cred *models.Credential, db *database.DB) *CredentialFormModel {
	inputs := make([]textinput.Model, fieldCount)

	for i := range inputs {
		inputs[i] = textinput.New()
		inputs[i].Width = 44
		inputs[i].PromptStyle = lipgloss.NewStyle().Foreground(styles.Primary)
		inputs[i].TextStyle = lipgloss.NewStyle().Foreground(styles.Text)
		inputs[i].PlaceholderStyle = lipgloss.NewStyle().Foreground(styles.Muted)
	}

	inputs[fieldName].Placeholder = "e.g., GitHub"
	inputs[fieldURL].Placeholder = "e.g., https://github.com/login"
	inputs[fieldUsername].Placeholder = "e.g., user@example.com"
	inputs[fieldPassword].Placeholder = "Enter password"
	inputs[fieldPassword].EchoMode = textinput.EchoPassword
	inputs[fieldPassword].EchoCharacter = '●'
	inputs[fieldNotes].Placeholder = "e.g., 2FA recovery codes"

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
	icon := "+"
	if m.editing != nil {
		title = "Edit Credential"
		icon = "~"
	}

	b.WriteString("\n")
	b.WriteString("  " + styles.TitleStyle.Render(icon+" "+title))
	b.WriteString("   " + styles.ProgressDots(m.focused, fieldCount))
	b.WriteString("\n\n")

	for i, meta := range credentialFields {
		b.WriteString(m.renderField(i, meta))
	}

	if m.err != "" {
		b.WriteString("\n")
		b.WriteString("  " + styles.DangerStyle.Render("! "+m.err))
		b.WriteString("\n")
	}

	return b.String()
}

func (m *CredentialFormModel) renderField(idx int, meta fieldMeta) string {
	var b strings.Builder

	// Label row with hint
	label := styles.LabelStyle.Render("  " + meta.label)
	hint := styles.HintStyle.Render(" (" + meta.hint + ")")
	b.WriteString(label + hint + "\n")

	// Description
	b.WriteString(styles.MutedStyle.Render("  "+meta.desc) + "\n")

	// Input with border that changes on focus
	inputView := m.inputs[idx].View()
	if idx == m.focused {
		b.WriteString(styles.InputGroupFocusedStyle.Render("  " + inputView))
	} else {
		b.WriteString(styles.InputGroupStyle.Render("  " + inputView))
	}
	b.WriteString("\n")

	return b.String()
}
