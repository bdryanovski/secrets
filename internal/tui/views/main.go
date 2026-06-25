package views

import (
	"fmt"
	"strings"

	"github.com/bdryanovski/secrets/internal/config"
	"github.com/bdryanovski/secrets/internal/database"
	"github.com/bdryanovski/secrets/internal/tui/styles"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

	statusMsg  string
	statusType string // "success", "error", "info"
}

// NewMainModel creates the main application model.
func NewMainModel(cfg *config.Config, db *database.DB, width, height int) *MainModel {
	m := &MainModel{
		cfg:       cfg,
		db:        db,
		width:     width,
		height:    height,
		activeTab: TabCredentials,
		viewMode:  ViewList,
	}
	m.credList = NewCredentialListModel(db)
	m.envList = NewEnvSecretListModel(db)
	// List sub-views are sized dynamically in View() based on
	// measured chrome height.
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
		// List sub-views are sized dynamically in View() based on
		// measured chrome height, so no SetSize call needed here.
		return m, nil

	case statusMsg:
		m.statusMsg = string(msg)
		m.statusType = "info"
		return m, nil

	case backToListMsg:
		m.viewMode = ViewList
		m.statusMsg = string(msg)
		m.statusType = "success"
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
		var cmd tea.Cmd
		if m.activeTab == TabCredentials && m.credForm != nil {
			cmd = m.credForm.handleKey(msg)
		} else if m.envForm != nil {
			cmd = m.envForm.handleKey(msg)
		}
		return m, cmd
	case "enter":
		var cmd tea.Cmd
		if m.activeTab == TabCredentials && m.credForm != nil {
			cmd = m.credForm.handleKey(msg)
		} else if m.envForm != nil {
			cmd = m.envForm.handleKey(msg)
		}
		return m, cmd
	}
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
				m.statusType = "success"
			}
		} else {
			env := m.envList.Selected()
			if env != nil {
				m.db.DeleteEnvSecret(env.ID)
				m.statusMsg = fmt.Sprintf("Deleted: %s [%s]", env.Key, env.Environment)
				m.statusType = "success"
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
		var cmd tea.Cmd
		if m.search != nil {
			cmd = m.search.handleKey(msg)
		}
		return m, cmd
	}
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
		var cmd tea.Cmd
		if m.importView != nil {
			cmd = m.importView.handleKey(msg)
		}
		return m, cmd
	}
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

// ── View ─────────────────────────────────────────────────────────────────────

func (m *MainModel) View() string {
	contentWidth := m.width
	if contentWidth < 60 {
		contentWidth = 60
	}
	if contentWidth > 100 {
		contentWidth = 100
	}

	// Render chrome sections first so we can measure their height.
	header := m.renderHeader(contentWidth)
	chromeHeight := lipgloss.Height(header)

	var tabs string
	if m.viewMode == ViewList || m.viewMode == ViewDelete {
		tabs = m.renderTabs(contentWidth)
		chromeHeight += lipgloss.Height(tabs)
	}

	footer := m.renderFooter()
	chromeHeight += lipgloss.Height(footer)

	var status string
	if m.statusMsg != "" {
		status = m.renderStatus(contentWidth)
		chromeHeight += lipgloss.Height(status)
	}

	// Give list sub-views exactly the remaining vertical space.
	listHeight := m.height - chromeHeight
	if listHeight < 3 {
		listHeight = 3
	}
	m.credList.SetSize(contentWidth, listHeight)
	m.envList.SetSize(contentWidth, listHeight)

	// Now render content with the correct size.
	content := m.renderContent(contentWidth)

	// Assemble all sections.
	var sections []string
	sections = append(sections, header)
	if tabs != "" {
		sections = append(sections, tabs)
	}
	sections = append(sections, content)
	if status != "" {
		sections = append(sections, status)
	}
	sections = append(sections, footer)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m *MainModel) renderHeader(width int) string {
	logo := styles.LogoSmall()
	spacer := strings.Repeat(" ", max(1, width-lipgloss.Width(logo)-20))
	viewLabel := m.viewModeLabel()

	header := styles.HeaderBarStyle.Width(width).Render(
		logo + spacer + styles.DimStyle.Render(viewLabel),
	)
	return header
}

func (m *MainModel) viewModeLabel() string {
	switch m.viewMode {
	case ViewList:
		return "Browse"
	case ViewDetail:
		return "Detail"
	case ViewAdd:
		return "Add New"
	case ViewEdit:
		return "Edit"
	case ViewDelete:
		return "Delete"
	case ViewSearch:
		return "Search"
	case ViewImport:
		return "Import"
	case ViewGeneratePassword:
		return "Generate"
	}
	return ""
}

func (m *MainModel) renderTabs(width int) string {
	var tabs []string
	tabLabels := []string{"  Credentials  ", "  Env Secrets  "}
	for i, label := range tabLabels {
		if i == m.activeTab {
			tabs = append(tabs, styles.ActiveTabStyle.Render(label))
		} else {
			tabs = append(tabs, styles.InactiveTabStyle.Render(label))
		}
	}
	tabRow := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	tabLine := styles.DividerStyle.Render(strings.Repeat("─", width))
	return tabRow + "\n" + tabLine
}

func (m *MainModel) renderContent(width int) string {
	switch m.viewMode {
	case ViewList:
		if m.activeTab == TabCredentials {
			return m.credList.View()
		}
		return m.envList.View()
	case ViewDetail:
		if m.detail != nil {
			return m.detail.View()
		}
	case ViewAdd, ViewEdit:
		if m.activeTab == TabCredentials && m.credForm != nil {
			return m.credForm.View()
		} else if m.envForm != nil {
			return m.envForm.View()
		}
	case ViewDelete:
		return m.renderDeleteConfirm()
	case ViewSearch:
		if m.search != nil {
			return m.search.View()
		}
	case ViewImport:
		if m.importView != nil {
			return m.importView.View()
		}
	case ViewGeneratePassword:
		if m.passGen != nil {
			return m.passGen.View()
		}
	}
	return ""
}

func (m *MainModel) renderStatus(width int) string {
	icon := "i"
	style := styles.StatusBarStyle
	switch m.statusType {
	case "success":
		icon = "+"
		style = style.Background(styles.Success).Foreground(styles.BgDark)
	case "error":
		icon = "!"
		style = style.Background(styles.Danger).Foreground(styles.Text)
	}
	return style.Width(width).Render(" " + icon + "  " + m.statusMsg)
}

func (m *MainModel) renderFooter() string {
	w := m.width
	switch m.viewMode {
	case ViewList:
		return styles.HelpBarWrapped(w,
			styles.KeyHint("tab", "switch"),
			styles.KeyHint("a", "add"),
			styles.KeyHint("enter", "view"),
			styles.KeyHint("d", "delete"),
			styles.KeyHint("/", "search"),
			styles.KeyHint("g", "generate"),
			styles.KeyHint("i", "import"),
			styles.KeyHint("q", "quit"),
		)
	case ViewDetail:
		return styles.HelpBarWrapped(w,
			styles.KeyHint("s", "show/hide"),
			styles.KeyHint("c", "copy"),
			styles.KeyHint("e", "edit"),
			styles.KeyHint("esc", "back"),
		)
	case ViewAdd, ViewEdit:
		return styles.HelpBarWrapped(w,
			styles.KeyHint("tab", "next field"),
			styles.KeyHint("shift+tab", "prev"),
			styles.KeyHint("ctrl+s", "save"),
			styles.KeyHint("esc", "cancel"),
		)
	case ViewDelete:
		return styles.HelpBarWrapped(w,
			styles.KeyHint("y", "confirm"),
			styles.KeyHint("n", "cancel"),
		)
	case ViewGeneratePassword:
		return styles.HelpBarWrapped(w,
			styles.KeyHint("r", "regenerate"),
			styles.KeyHint("c", "copy"),
			styles.KeyHint("+/-", "length"),
			styles.KeyHint("u", "upper"),
			styles.KeyHint("l", "lower"),
			styles.KeyHint("d", "digits"),
			styles.KeyHint("s", "symbols"),
			styles.KeyHint("esc", "back"),
		)
	case ViewSearch:
		return styles.HelpBarWrapped(w,
			styles.KeyHint("enter", "search"),
			styles.KeyHint("esc", "back"),
		)
	case ViewImport:
		return styles.HelpBarWrapped(w,
			styles.KeyHint("1/2", "source"),
			styles.KeyHint("enter", "import"),
			styles.KeyHint("esc", "back"),
		)
	default:
		return styles.HelpBarWrapped(w, styles.KeyHint("esc", "back"))
	}
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

	content := lipgloss.JoinVertical(lipgloss.Left,
		"",
		styles.DangerStyle.Render("  Are you sure you want to delete this?"),
		"",
		"  "+styles.HeadingStyle.Render(name),
		"",
		"  "+styles.MutedStyle.Render("This action cannot be undone."),
		"",
		"  Press "+styles.KeyStyle.Render("y")+" to confirm, "+
			styles.KeyStyle.Render("n")+" to cancel",
		"",
	)
	return "\n" + styles.DangerCardStyle.Render(content)
}

// statusMsg is a tea.Msg carrying a status bar message.
type statusMsg string

// backToListMsg signals the main view to return to list mode.
type backToListMsg string
