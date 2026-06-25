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
	envFieldKey = iota
	envFieldValue
	envFieldEnvironment
	envFieldDescription
	envFieldCount
)

// EnvSecretFormModel handles adding/editing an env secret.
type EnvSecretFormModel struct {
	db      *database.DB
	editing *models.EnvSecret // nil if adding new
	inputs  []textinput.Model
	focused int
	err     string
}

// NewEnvSecretFormModel creates an env secret form. Pass nil for a new secret.
func NewEnvSecretFormModel(env *models.EnvSecret, db *database.DB) *EnvSecretFormModel {
	inputs := make([]textinput.Model, envFieldCount)

	for i := range inputs {
		inputs[i] = textinput.New()
		inputs[i].Width = 40
	}

	inputs[envFieldKey].Placeholder = "Key (e.g., API_KEY)"
	inputs[envFieldValue].Placeholder = "Value"
	inputs[envFieldValue].EchoMode = textinput.EchoPassword
	inputs[envFieldValue].EchoCharacter = '*'
	inputs[envFieldEnvironment].Placeholder = "Environment (development/staging/production)"
	inputs[envFieldDescription].Placeholder = "Description (optional)"

	if env != nil {
		inputs[envFieldKey].SetValue(env.Key)
		inputs[envFieldValue].SetValue(env.Value)
		inputs[envFieldEnvironment].SetValue(env.Environment)
		inputs[envFieldDescription].SetValue(env.Description)
	} else {
		inputs[envFieldEnvironment].SetValue("development")
	}

	inputs[envFieldKey].Focus()

	return &EnvSecretFormModel{
		db:      db,
		editing: env,
		inputs:  inputs,
	}
}

func (m *EnvSecretFormModel) update(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd
	for i := range m.inputs {
		var cmd tea.Cmd
		m.inputs[i], cmd = m.inputs[i].Update(msg)
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

func (m *EnvSecretFormModel) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "tab", "down":
		m.inputs[m.focused].Blur()
		m.focused = (m.focused + 1) % envFieldCount
		m.inputs[m.focused].Focus()
		return textinput.Blink
	case "shift+tab", "up":
		m.inputs[m.focused].Blur()
		m.focused = (m.focused - 1 + envFieldCount) % envFieldCount
		m.inputs[m.focused].Focus()
		return textinput.Blink
	case "enter":
		if m.focused == envFieldCount-1 {
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

func (m *EnvSecretFormModel) save() tea.Cmd {
	key := strings.TrimSpace(m.inputs[envFieldKey].Value())
	if key == "" {
		m.err = "Key is required"
		return nil
	}

	env := strings.TrimSpace(m.inputs[envFieldEnvironment].Value())
	if env == "" {
		env = "development"
	}

	secret := &models.EnvSecret{
		Key:         key,
		Value:       m.inputs[envFieldValue].Value(),
		Environment: env,
		Description: strings.TrimSpace(m.inputs[envFieldDescription].Value()),
	}

	db := m.db
	editing := m.editing

	return func() tea.Msg {
		if editing != nil {
			secret.ID = editing.ID
			if err := db.UpdateEnvSecret(secret); err != nil {
				return statusMsg("Error: " + err.Error())
			}
			return backToListMsg("Updated: " + secret.Key + " [" + secret.Environment + "]")
		}

		if err := db.CreateEnvSecret(secret); err != nil {
			return statusMsg("Error: " + err.Error())
		}
		return backToListMsg("Added: " + secret.Key + " [" + secret.Environment + "]")
	}
}

func (m *EnvSecretFormModel) View() string {
	var b strings.Builder

	title := "Add Env Secret"
	if m.editing != nil {
		title = "Edit Env Secret"
	}
	b.WriteString("\n")
	b.WriteString(styles.TitleStyle.Render("  " + title))
	b.WriteString("\n\n")

	labels := []string{"Key:", "Value:", "Environment:", "Description:"}
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
