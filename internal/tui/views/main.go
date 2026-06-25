package views

import (
	"fmt"
	"strings"

	"github.com/bdryanovski/secrets/internal/config"
	"github.com/bdryanovski/secrets/internal/database"
	"github.com/bdryanovski/secrets/internal/tui/styles"
	"github.com/bdryanovski/secrets/internal/version"

	tea "github.com/charmbracelet/bubbletea"
)

// Tab constants
const (
	TabCredentials = iota
	TabEnvSecrets
)

// ViewMode determines the current view state.
type ViewMode int

const (
	ViewList ViewMode = iota
	ViewDetail
	ViewAdd
	ViewEdit
	ViewDelete
	ViewSearch
	ViewImport
	ViewGeneratePassword
)

// MainModel is the primary TUI model after unlock.
type MainModel struct {
	cfg    *config.Config
	db     *database.DB
	width  int
	height int

	activeTab int
	viewMode  ViewMode

	// Sub-views
	credList   *CredentialListModel
	envList    *EnvSecretListModel
	credForm   *CredentialFormModel
	envForm    *EnvSecretFormModel
	detail     *DetailModel
	search     *SearchModel
	importView *ImportModel
	passGen    *PasswordGenModel

	statusMsg string
}

// NewMainModel creates the main application model.
func NewMainModel(cfg *config.Config, db *database.DB) *MainModel {
	m := &MainModel{
		cfg:       cfg,
		db:        db,
		activeTab: TabCredentials,
		viewMode:  ViewList,
	}

	m.credList = NewCredentialListModel(db)
	m.envList = NewEnvSecretListModel(db)

	return m
}

func (m *MainModel) Init() tea.Cmd {
	return tea.Batch(
		m.credList.loadCredentials(),
		m.envList.loadEnvSecrets(),
	)
}

func (m *MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case statusMsg:
		m.statusMsg = string(msg)
		return m, nil

	case backToListMsg:
		m.viewMode = ViewList
		m.statusMsg = string(msg)
		var cmd tea.Cmd
		if m.activeTab == TabCredentials {
			cmd = m.credList.loadCredentials()
		} else {
			cmd = m.envList.loadEnvSecrets()
		}
		return m, cmd

	case tea.KeyMsg:
		// Global keys
		switch msg.String() {
		case "ctrl+c":
			m.db.Close()
			return m, tea.Quit
		case "q":
			if m.viewMode == ViewList {
				m.db.Close()
				return m, tea.Quit
			}
		}

		// Handle based on view mode.
		switch m.viewMode {
		case ViewList:
			return m.handleListKeys(msg)
		case ViewDetail:
			return m.handleDetailKeys(msg)
		case ViewAdd, ViewEdit:
			return m.handleFormKeys(msg)
		case ViewDelete:
			return m.handleDeleteKeys(msg)
		case ViewSearch:
			return m.handleSearchKeys(msg)
		case ViewImport:
			return m.handleImportKeys(msg)
		case ViewGeneratePassword:
			return m.handlePassGenKeys(msg)
		}
	}

	// Delegate to sub-views.
	return m.delegateUpdate(msg)
}

func (m *MainModel) handleListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		if m.activeTab == TabCredentials {
			m.activeTab = TabEnvSecrets
		} else {
			m.activeTab = TabCredentials
		}
		return m, nil

	case "a":
		m.viewMode = ViewAdd
		if m.activeTab == TabCredentials {
			m.credForm = NewCredentialFormModel(nil, m.db)
		} else {
			m.envForm = NewEnvSecretFormModel(nil, m.db)
		}
		return m, nil

	case "enter":
		m.viewMode = ViewDetail
		if m.activeTab == TabCredentials {
			cred := m.credList.Selected()
			if cred != nil {
				m.detail = NewDetailModel(m.db, m.cfg, cred.ID, DetailCredential)
				return m, m.detail.load()
			}
		} else {
			env := m.envList.Selected()
			if env != nil {
				m.detail = NewDetailModel(m.db, m.cfg, env.ID, DetailEnvSecret)
				return m, m.detail.load()
			}
		}
		m.viewMode = ViewList
		return m, nil

	case "d":
		m.viewMode = ViewDelete
		return m, nil

	case "/":
		m.viewMode = ViewSearch
		m.search = NewSearchModel(m.db)
		return m, m.search.Init()

	case "i":
		m.viewMode = ViewImport
		m.importView = NewImportModel(m.db)
		return m, nil

	case "g":
		m.viewMode = ViewGeneratePassword
		m.passGen = NewPasswordGenModel(m.cfg)
		return m, m.passGen.Init()
	}

	// Delegate navigation to list sub-views.
	var cmd tea.Cmd
	if m.activeTab == TabCredentials {
		cmd = m.credList.handleKey(msg)
	} else {
		cmd = m.envList.handleKey(msg)
	}
	return m, cmd
}

func (m *MainModel) handleDetailKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "backspace":
		m.viewMode = ViewList
		return m, nil
	case "e":
		m.viewMode = ViewEdit
		if m.detail.detailType == DetailCredential {
			cred, _ := m.db.GetCredential(m.detail.itemID)
			if cred != nil {
				m.credForm = NewCredentialFormModel(cred, m.db)
			}
		} else {
			env, _ := m.db.GetEnvSecret(m.detail.itemID)
			if env != nil {
				m.envForm = NewEnvSecretFormModel(env, m.db)
			}
		}
		return m, nil
	}

	var cmd tea.Cmd
	if m.detail != nil {
		cmd = m.detail.handleKey(msg)
	}
	return m, cmd
}

func (m *MainModel) handleFormKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.viewMode = ViewList
		return m, nil
	case "tab", "shift+tab", "up", "down", "ctrl+s":
		// Navigation and save keys -- handled by the form's handleKey only.
		var cmd tea.Cmd
		if m.activeTab == TabCredentials && m.credForm != nil {
			cmd = m.credForm.handleKey(msg)
		} else if m.envForm != nil {
			cmd = m.envForm.handleKey(msg)
		}
		return m, cmd
	case "enter":
		// Enter moves to next field or saves -- handled by handleKey only.
		var cmd tea.Cmd
		if m.activeTab == TabCredentials && m.credForm != nil {
			cmd = m.credForm.handleKey(msg)
		} else if m.envForm != nil {
			cmd = m.envForm.handleKey(msg)
		}
		return m, cmd
	}

	// All other keys (character input) -- forward to textinput via update.
	var cmd tea.Cmd
	if m.activeTab == TabCredentials && m.credForm != nil {
		cmd = m.credForm.update(msg)
	} else if m.envForm != nil {
		cmd = m.envForm.update(msg)
	}
	return m, cmd
}

func (m *MainModel) handleDeleteKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if m.activeTab == TabCredentials {
			cred := m.credList.Selected()
			if cred != nil {
				m.db.DeleteCredential(cred.ID)
				m.statusMsg = fmt.Sprintf("Deleted: %s", cred.Name)
			}
		} else {
			env := m.envList.Selected()
			if env != nil {
				m.db.DeleteEnvSecret(env.ID)
				m.statusMsg = fmt.Sprintf("Deleted: %s [%s]", env.Key, env.Environment)
			}
		}
		m.viewMode = ViewList
		var cmd tea.Cmd
		if m.activeTab == TabCredentials {
			cmd = m.credList.loadCredentials()
		} else {
			cmd = m.envList.loadEnvSecrets()
		}
		return m, cmd
	case "n", "N", "esc":
		m.viewMode = ViewList
		return m, nil
	}
	return m, nil
}

func (m *MainModel) handleSearchKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.viewMode = ViewList
		return m, nil
	case "enter":
		// Enter triggers search -- handled by handleKey only.
		var cmd tea.Cmd
		if m.search != nil {
			cmd = m.search.handleKey(msg)
		}
		return m, cmd
	}

	// Character input -- forward to textinput via update.
	var cmd tea.Cmd
	if m.search != nil {
		cmd = m.search.update(msg)
	}
	return m, cmd
}

func (m *MainModel) handleImportKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.viewMode = ViewList
		return m, nil
	case "enter", "1", "2":
		// Enter triggers import/save, 1/2 selects source -- handled by handleKey only.
		var cmd tea.Cmd
		if m.importView != nil {
			cmd = m.importView.handleKey(msg)
		}
		return m, cmd
	}

	// Character input -- forward to textinput via update.
	var cmd tea.Cmd
	if m.importView != nil {
		cmd = m.importView.update(msg)
	}
	return m, cmd
}

func (m *MainModel) handlePassGenKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.viewMode = ViewList
		return m, nil
	}

	var cmd tea.Cmd
	if m.passGen != nil {
		cmd = m.passGen.handleKey(msg)
	}
	return m, cmd
}

func (m *MainModel) delegateUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.viewMode {
	case ViewList:
		if m.activeTab == TabCredentials {
			cmd = m.credList.update(msg)
		} else {
			cmd = m.envList.update(msg)
		}
	case ViewDetail:
		if m.detail != nil {
			cmd = m.detail.update(msg)
		}
	case ViewAdd, ViewEdit:
		if m.activeTab == TabCredentials && m.credForm != nil {
			cmd = m.credForm.update(msg)
		} else if m.envForm != nil {
			cmd = m.envForm.update(msg)
		}
	case ViewSearch:
		if m.search != nil {
			cmd = m.search.update(msg)
		}
	case ViewImport:
		if m.importView != nil {
			cmd = m.importView.update(msg)
		}
	case ViewGeneratePassword:
		if m.passGen != nil {
			cmd = m.passGen.update(msg)
		}
	}
	return m, cmd
}

func (m *MainModel) View() string {
	var b strings.Builder

	// Header
	b.WriteString(styles.TitleStyle.Render("  Secrets Manager"))
	b.WriteString("  ")
	b.WriteString(styles.MutedStyle.Render(version.Version))
	b.WriteString("\n")

	// Tabs
	b.WriteString("  ")
	if m.activeTab == TabCredentials {
		b.WriteString(styles.ActiveTabStyle.Render("Credentials"))
	} else {
		b.WriteString(styles.InactiveTabStyle.Render("Credentials"))
	}
	if m.activeTab == TabEnvSecrets {
		b.WriteString(styles.ActiveTabStyle.Render("Env Secrets"))
	} else {
		b.WriteString(styles.InactiveTabStyle.Render("Env Secrets"))
	}
	b.WriteString("\n")
	b.WriteString(styles.MutedStyle.Render("  " + strings.Repeat("─", 50)))
	b.WriteString("\n")

	// Content
	switch m.viewMode {
	case ViewList:
		if m.activeTab == TabCredentials {
			b.WriteString(m.credList.View())
		} else {
			b.WriteString(m.envList.View())
		}
	case ViewDetail:
		if m.detail != nil {
			b.WriteString(m.detail.View())
		}
	case ViewAdd, ViewEdit:
		if m.activeTab == TabCredentials && m.credForm != nil {
			b.WriteString(m.credForm.View())
		} else if m.envForm != nil {
			b.WriteString(m.envForm.View())
		}
	case ViewDelete:
		b.WriteString(m.renderDeleteConfirm())
	case ViewSearch:
		if m.search != nil {
			b.WriteString(m.search.View())
		}
	case ViewImport:
		if m.importView != nil {
			b.WriteString(m.importView.View())
		}
	case ViewGeneratePassword:
		if m.passGen != nil {
			b.WriteString(m.passGen.View())
		}
	}

	// Status bar
	if m.statusMsg != "" {
		b.WriteString("\n")
		b.WriteString(styles.StatusBarStyle.Render("  " + m.statusMsg))
	}

	// Help
	b.WriteString("\n")
	if m.viewMode == ViewList {
		b.WriteString(styles.HelpStyle.Render("  tab: switch  a: add  enter: view  d: delete  /: search  g: generate  i: import  q: quit"))
	}

	return b.String()
}

func (m *MainModel) renderDeleteConfirm() string {
	var name string
	if m.activeTab == TabCredentials {
		cred := m.credList.Selected()
		if cred != nil {
			name = cred.Name
		}
	} else {
		env := m.envList.Selected()
		if env != nil {
			name = fmt.Sprintf("%s [%s]", env.Key, env.Environment)
		}
	}

	return styles.BoxStyle.Render(
		styles.DangerStyle.Render("  Delete: "+name+"?") + "\n\n" +
			"  Press " + styles.DangerStyle.Render("y") + " to confirm, " +
			styles.MutedStyle.Render("n") + " to cancel",
	)
}

// statusMsg is a tea.Msg carrying a status bar message.
type statusMsg string

// backToListMsg signals the main view to return to list mode.
type backToListMsg string
