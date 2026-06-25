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

// EnvSecretListModel displays a list of environment secrets.
type EnvSecretListModel struct {
	db     *database.DB
	items  []models.EnvSecret
	cursor int
	offset int // first visible row index
	err    string
	filter string
	width  int
	height int // available rows for the list content
}

// NewEnvSecretListModel creates a new env secret list view.
func NewEnvSecretListModel(db *database.DB) *EnvSecretListModel {
	return &EnvSecretListModel{db: db}
}

// SetSize updates the available dimensions for the list.
func (m *EnvSecretListModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// visibleRows returns how many data rows fit in the available height.
// Conservative estimate for keyboard navigation (before View renders).
func (m *EnvSecretListModel) visibleRows() int {
	// pills(1) + badge(1) + header(1) + divider(1) + scroll indicators(2) + padding(2) = 8
	rows := m.height - 8
	if rows < 1 {
		rows = 1
	}
	return rows
}

// ensureCursorVisible adjusts the scroll offset so the cursor is always in view.
func (m *EnvSecretListModel) ensureCursorVisible() {
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

type envSecretsLoadedMsg struct {
	items []models.EnvSecret
	err   error
}

func (m *EnvSecretListModel) loadEnvSecrets() tea.Cmd {
	db := m.db
	filter := m.filter
	return func() tea.Msg {
		items, err := db.ListEnvSecrets(filter)
		return envSecretsLoadedMsg{items: items, err: err}
	}
}

func (m *EnvSecretListModel) update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case envSecretsLoadedMsg:
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

func (m *EnvSecretListModel) handleKey(msg tea.KeyMsg) tea.Cmd {
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
	case "1":
		m.filter = "development"
		m.cursor = 0
		m.offset = 0
		return m.loadEnvSecrets()
	case "2":
		m.filter = "staging"
		m.cursor = 0
		m.offset = 0
		return m.loadEnvSecrets()
	case "3":
		m.filter = "production"
		m.cursor = 0
		m.offset = 0
		return m.loadEnvSecrets()
	case "0":
		m.filter = ""
		m.cursor = 0
		m.offset = 0
		return m.loadEnvSecrets()
	}
	m.ensureCursorVisible()
	return nil
}

// Selected returns the currently selected env secret, or nil.
func (m *EnvSecretListModel) Selected() *models.EnvSecret {
	if m.cursor >= 0 && m.cursor < len(m.items) {
		return &m.items[m.cursor]
	}
	return nil
}

func (m *EnvSecretListModel) View() string {
	// Environment filter pills (always shown)
	var pills strings.Builder
	pills.WriteString("  ")
	filters := []struct {
		key, label, value string
	}{
		{"0", "All", ""},
		{"1", "Dev", "development"},
		{"2", "Staging", "staging"},
		{"3", "Prod", "production"},
	}
	for i, f := range filters {
		if i > 0 {
			pills.WriteString(" ")
		}
		label := f.label + " (" + f.key + ")"
		if m.filter == f.value {
			pills.WriteString(styles.Badge(" "+label+" ", styles.BgDark, styles.Primary))
		} else {
			pills.WriteString(styles.Badge(" "+label+" ", styles.TextDim, styles.BgCard))
		}
	}
	pillsStr := pills.String()

	if m.err != "" {
		return pillsStr + "\n\n" + styles.DangerCardStyle.Render(
			styles.DangerStyle.Render("  Error: "+m.err),
		) + "\n"
	}

	if len(m.items) == 0 {
		// Pills take 1 line + newline = 2 lines of overhead
		pillsHeight := lipgloss.Height(pillsStr) + 1
		emptyHeight := m.height - pillsHeight
		if emptyHeight < 3 {
			emptyHeight = 3
		}
		return pillsStr + "\n" + m.emptyStateWithHeight(emptyHeight)
	}

	// Determine column widths based on available width
	rowWidth := m.width - 4
	if rowWidth < 40 {
		rowWidth = 40
	}
	keyW := rowWidth * 38 / 100
	descW := rowWidth * 25 / 100
	if keyW < 10 {
		keyW = 10
	}
	if descW < 8 {
		descW = 8
	}

	// Build top chrome so we can measure it.
	countBadge := styles.Badge(
		fmt.Sprintf(" %d items ", len(m.items)),
		styles.BgDark, styles.PrimaryDim,
	)

	//  "   " + num(3) + " " + key(keyW) + " " + env(8) + " " + desc
	headerLeft := "   " + padRight("#", 3) + " " + padRight("KEY", keyW) + " " + padRight("ENV", 8) + " " + "DESCRIPTION"
	headerLeftW := lipgloss.Width(headerLeft)
	badgeW := lipgloss.Width(countBadge)
	gap := m.width - headerLeftW - badgeW
	if gap < 1 {
		gap = 1
	}
	header := styles.MutedStyle.Render(headerLeft) + strings.Repeat(" ", gap) + countBadge

	dividerW := min(keyW+descW+20, rowWidth)

	var top strings.Builder
	top.WriteString(pillsStr + "\n")
	top.WriteString(header + "\n")
	top.WriteString("  " + styles.Divider(dividerW))
	topStr := top.String()

	// Measure chrome height: top chrome + 1 scroll-up + 1 scroll-down.
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
		b.WriteString(m.renderRow(i, m.items[i], keyW, descW) + "\n")
	}

	// Scroll-down indicator
	remaining := len(m.items) - end
	if remaining > 0 {
		b.WriteString(styles.MutedStyle.Render(fmt.Sprintf("  ▼ %d more below", remaining)))
	}

	return b.String()
}

func (m *EnvSecretListModel) renderRow(idx int, env models.EnvSecret, keyW, descW int) string {
	num := padRight(fmt.Sprintf("%d", idx+1), 3)
	key := padRight(truncate(env.Key, keyW), keyW)
	badge := styles.EnvBadge(env.Environment)
	badgePadded := padRight(badge, 8) // badges are max ~6 visual chars, pad to 8
	desc := truncate(env.Description, descW)

	totalW := m.width
	if totalW < 40 {
		totalW = 40
	}

	if idx == m.cursor {
		// Selected: full-width highlighted background
		row := " " + styles.SelectedStyle.Render("▸") + " " + num + " " + key + " " + badgePadded + " " + desc
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
		styles.NormalStyle.Render(key) + " " +
		badgePadded + " " +
		styles.MutedStyle.Render(desc)
}

func (m *EnvSecretListModel) emptyStateWithHeight(h int) string {
	art := lipgloss.NewStyle().Foreground(styles.Subtle).Render(`
      _____
     |     |
     | ENV |
     |_____|
      |   |
      |___|
`)

	content := lipgloss.JoinVertical(lipgloss.Center,
		art,
		styles.MutedStyle.Render("No environment secrets stored yet"),
		"",
		styles.DimStyle.Render("Press ")+styles.KeyStyle.Render("a")+styles.DimStyle.Render(" to add your first env secret"),
		styles.DimStyle.Render("Secrets can have different values per environment"),
	)
	return lipgloss.Place(m.width, h, lipgloss.Center, lipgloss.Center, content)
}
