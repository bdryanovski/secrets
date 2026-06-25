package views

import (
	"fmt"
	"strings"

	"github.com/bdryanovski/secrets/internal/database"
	"github.com/bdryanovski/secrets/internal/models"
	"github.com/bdryanovski/secrets/internal/tui/styles"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CredentialListModel displays a list of credentials.
type CredentialListModel struct {
	db     *database.DB
	items  []models.Credential
	cursor int
	offset int // first visible row index
	err    string
	width  int
	height int // available rows for the list content
}

// NewCredentialListModel creates a new credential list view.
func NewCredentialListModel(db *database.DB) *CredentialListModel {
	return &CredentialListModel{db: db}
}

// SetSize updates the available dimensions for the list.
func (m *CredentialListModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// visibleRows returns how many data rows fit in the available height.
// Uses chromeLines (computed in View) if available, otherwise estimates.
func (m *CredentialListModel) visibleRows() int {
	// Conservative estimate for keyboard navigation (before View renders).
	// badge area(3) + table header(1) + divider(1) + scroll indicators(2) + padding(1) = 8
	rows := m.height - 8
	if rows < 1 {
		rows = 1
	}
	return rows
}

// ensureCursorVisible adjusts the scroll offset so the cursor is always in view.
func (m *CredentialListModel) ensureCursorVisible() {
	visible := m.visibleRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+visible {
		m.offset = m.cursor - visible + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

type credentialsLoadedMsg struct {
	items []models.Credential
	err   error
}

func (m *CredentialListModel) loadCredentials() tea.Cmd {
	db := m.db
	return func() tea.Msg {
		items, err := db.ListCredentials()
		return credentialsLoadedMsg{items: items, err: err}
	}
}

func (m *CredentialListModel) update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case credentialsLoadedMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.items = msg.items
			m.err = ""
			if m.cursor >= len(m.items) {
				m.cursor = max(0, len(m.items)-1)
			}
			m.ensureCursorVisible()
		}
	}
	return nil
}

func (m *CredentialListModel) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}
	case "home":
		m.cursor = 0
	case "end":
		if len(m.items) > 0 {
			m.cursor = len(m.items) - 1
		}
	case "pgup":
		visible := m.visibleRows()
		m.cursor -= visible
		if m.cursor < 0 {
			m.cursor = 0
		}
	case "pgdown":
		visible := m.visibleRows()
		m.cursor += visible
		if m.cursor >= len(m.items) {
			m.cursor = len(m.items) - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
	}
	m.ensureCursorVisible()
	return nil
}

// Selected returns the currently selected credential, or nil.
func (m *CredentialListModel) Selected() *models.Credential {
	if m.cursor >= 0 && m.cursor < len(m.items) {
		return &m.items[m.cursor]
	}
	return nil
}

func (m *CredentialListModel) View() string {
	if m.err != "" {
		errCard := styles.DangerCardStyle.Render(
			styles.DangerStyle.Render("  Error: " + m.err),
		)
		return "\n" + errCard + "\n"
	}

	if len(m.items) == 0 {
		return m.emptyState()
	}

	// Determine column widths based on available width
	rowWidth := m.width - 4
	if rowWidth < 40 {
		rowWidth = 40
	}
	nameW := rowWidth * 35 / 100
	userW := rowWidth * 30 / 100
	urlW := rowWidth * 25 / 100
	if nameW < 10 {
		nameW = 10
	}
	if userW < 8 {
		userW = 8
	}
	if urlW < 8 {
		urlW = 8
	}

	// Build chrome (everything above and below the data rows).
	countText := fmt.Sprintf(" %d items ", len(m.items))
	countBadge := styles.Badge(countText, styles.BgDark, styles.PrimaryDim)

	//  "   " + num(3) + " " + name(nameW) + " " + user(userW) + " " + url
	headerLeft := "   " + padRight("#", 3) + " " + padRight("NAME", nameW) + " " + padRight("USERNAME", userW) + " " + "URL"
	headerLeftW := lipgloss.Width(headerLeft)
	badgeW := lipgloss.Width(countBadge)
	gap := m.width - headerLeftW - badgeW
	if gap < 1 {
		gap = 1
	}
	header := styles.MutedStyle.Render(headerLeft) + strings.Repeat(" ", gap) + countBadge

	dividerW := min(nameW+userW+urlW+10, rowWidth)

	// Render top chrome into a string so we can measure it.
	var top strings.Builder
	top.WriteString(header + "\n")
	top.WriteString("  " + styles.Divider(dividerW))
	topStr := top.String()

	// Measure chrome height: top chrome + 1 line for scroll-up + 1 line for scroll-down.
	chromeLines := lipgloss.Height(topStr) + 2

	// Compute how many data rows fit.
	visible := m.height - chromeLines
	if visible < 1 {
		visible = 1
	}

	m.ensureCursorVisible()

	end := m.offset + visible
	if end > len(m.items) {
		end = len(m.items)
	}

	// Assemble the output.
	var b strings.Builder
	b.WriteString(topStr + "\n")

	// Scroll-up indicator
	if m.offset > 0 {
		b.WriteString(styles.MutedStyle.Render(fmt.Sprintf("  ▲ %d more above", m.offset)))
	}
	b.WriteString("\n")

	for i := m.offset; i < end; i++ {
		b.WriteString(m.renderRow(i, m.items[i], nameW, userW, urlW) + "\n")
	}

	// Scroll-down indicator
	remaining := len(m.items) - end
	if remaining > 0 {
		b.WriteString(styles.MutedStyle.Render(fmt.Sprintf("  ▼ %d more below", remaining)))
	}

	return b.String()
}

func (m *CredentialListModel) renderRow(idx int, cred models.Credential, nameW, userW, urlW int) string {
	num := padRight(fmt.Sprintf("%d", idx+1), 3)
	name := padRight(truncate(cred.Name, nameW), nameW)
	user := padRight(truncate(cred.Username, userW), userW)
	url := truncate(cred.URL, urlW)

	totalW := m.width
	if totalW < 40 {
		totalW = 40
	}

	if idx == m.cursor {
		// Selected: full-width highlighted background
		row := " " + styles.SelectedStyle.Render("▸") + " " + num + " " + name + " " + user + " " + url
		// Pad the row to full width so the background fills the line
		rowW := lipgloss.Width(row)
		if rowW < totalW {
			row += strings.Repeat(" ", totalW-rowW)
		}
		return lipgloss.NewStyle().
			Foreground(styles.Text).
			Background(styles.BgSurface).
			Bold(true).
			MaxWidth(totalW).
			Render(row)
	}

	return "   " +
		styles.MutedStyle.Render(num) + " " +
		styles.NormalStyle.Render(name) + " " +
		styles.DimStyle.Render(user) + " " +
		styles.MutedStyle.Render(url)
}

// padRight pads s with spaces to exactly width visual columns.
// If s is already wider, it is returned as-is.
func padRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func (m *CredentialListModel) emptyState() string {
	art := lipgloss.NewStyle().Foreground(styles.Subtle).Render(`
       ___
      |   |
      |   |
      |___|
     /     \
    / () () \
   |   __   |
    \_______/
`)

	content := lipgloss.JoinVertical(lipgloss.Center,
		art,
		styles.MutedStyle.Render("No credentials stored yet"),
		"",
		styles.DimStyle.Render("Press ")+styles.KeyStyle.Render("a")+styles.DimStyle.Render(" to add your first credential"),
		styles.DimStyle.Render("or ")+styles.KeyStyle.Render("i")+styles.DimStyle.Render(" to import from Bitwarden / Apple"),
	)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func truncate(s string, maxLen int) string {
	if maxLen <= 3 {
		if maxLen <= 0 {
			return ""
		}
		return s[:min(len(s), maxLen)]
	}
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}
