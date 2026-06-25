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
	envFieldKey = iota
	envFieldValue
	envFieldEnvironment
	envFieldDescription
	envFieldCount
)

var envSecretFields = []fieldMeta{
	{
		label: "Key",
		hint:  "Required",
		desc:  "The environment variable name (e.g., API_KEY, DATABASE_URL)",
	},
	{
		label: "Value",
		hint:  "Will be encrypted",
		desc:  "The secret value, encrypted with AES-256-GCM before storage",
	},
	{
		label: "Environment",
		hint:  "development / staging / production",
		desc:  "Which environment this value belongs to. Same key can have different values per env",
	},
	{
		label: "Description",
		hint:  "Optional",
		desc:  "A note about what this secret is used for or where it comes from",
	},
}

// EnvSecretFormModel handles adding/editing an env secret.
type EnvSecretFormModel struct {
	db      *database.DB
	editing *models.EnvSecret
	inputs  []textinput.Model
	focused int
	err     string
}

// NewEnvSecretFormModel creates an env secret form.
func NewEnvSecretFormModel(env *models.EnvSecret, db *database.DB) *EnvSecretFormModel {
	inputs := make([]textinput.Model, envFieldCount)

	for i := range inputs {
		inputs[i] = textinput.New()
		inputs[i].Width = 44
		inputs[i].PromptStyle = lipgloss.NewStyle().Foreground(styles.Primary)
		inputs[i].TextStyle = lipgloss.NewStyle().Foreground(styles.Text)
		inputs[i].PlaceholderStyle = lipgloss.NewStyle().Foreground(styles.Muted)
	}

	inputs[envFieldKey].Placeholder = "e.g., API_KEY"
	inputs[envFieldValue].Placeholder = "Enter secret value"
	inputs[envFieldValue].EchoMode = textinput.EchoPassword
	inputs[envFieldValue].EchoCharacter = '●'
	inputs[envFieldEnvironment].Placeholder = "development"
	inputs[envFieldDescription].Placeholder = "e.g., Stripe API key for payments"

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
	icon := "+"
	if m.editing != nil {
		title = "Edit Env Secret"
		icon = "~"
	}

	b.WriteString("\n")
	b.WriteString("  " + styles.TitleStyle.Render(icon+" "+title))
	b.WriteString("   " + styles.ProgressDots(m.focused, envFieldCount))
	b.WriteString("\n")
	b.WriteString("  " + styles.MutedStyle.Render("Environment secrets can be sourced into your shell"))
	b.WriteString("\n")
	b.WriteString("  " + styles.MutedStyle.Render("using: eval $(secrets env --profile <environment>)"))
	b.WriteString("\n\n")

	for i, meta := range envSecretFields {
		b.WriteString(m.renderField(i, meta))
	}

	if m.err != "" {
		b.WriteString("\n")
		b.WriteString("  " + styles.DangerStyle.Render("! "+m.err))
		b.WriteString("\n")
	}

	return b.String()
}

func (m *EnvSecretFormModel) renderField(idx int, meta fieldMeta) string {
	var b strings.Builder

	label := styles.LabelStyle.Render("  " + meta.label)
	hint := styles.HintStyle.Render(" (" + meta.hint + ")")
	b.WriteString(label + hint + "\n")

	b.WriteString(styles.MutedStyle.Render("  "+meta.desc) + "\n")

	inputView := m.inputs[idx].View()
	if idx == m.focused {
		b.WriteString(styles.InputGroupFocusedStyle.Render("  " + inputView))
	} else {
		b.WriteString(styles.InputGroupStyle.Render("  " + inputView))
	}
	b.WriteString("\n")

	return b.String()
}
