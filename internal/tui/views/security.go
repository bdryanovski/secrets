package views

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/bdryanovski/secrets/internal/database"
	"github.com/bdryanovski/secrets/internal/models"
	"github.com/bdryanovski/secrets/internal/tui/styles"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// securityTab identifies the active section.
type securityTab int

const (
	tabOverview securityTab = iota
	tabDuplicates
	tabIdentities
)

// securityEditMsg signals MainModel to edit a credential from security view.
type securityEditMsg struct {
	credID int64
}

// SecurityModel displays security analysis of the vault.
type SecurityModel struct {
	db     *database.DB
	tab    securityTab
	scroll int
	cursor int // cursor within selectableItems
	height int
	width  int

	// Analysis results
	totalCreds     int
	emptyPasswords []models.Credential
	duplicates     map[string][]models.Credential // password hash -> credentials sharing it
	identities     map[string][]models.Credential // username -> credentials using it
	sortedDupKeys  []string                       // sorted keys for stable display
	sortedIdKeys   []string                       // sorted keys for stable display
	loaded         bool
	err            string

	// Selectable items for the current tab
	selectableItems []models.Credential
}

// NewSecurityModel creates a security analysis view.
func NewSecurityModel(db *database.DB) *SecurityModel {
	return &SecurityModel{db: db}
}

type securityLoadedMsg struct {
	creds []models.Credential
	err   error
}

func (m *SecurityModel) Init() tea.Cmd {
	db := m.db
	return func() tea.Msg {
		creds, err := db.ListCredentialsDecrypted()
		return securityLoadedMsg{creds: creds, err: err}
	}
}

// SetSize updates the available dimensions.
func (m *SecurityModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *SecurityModel) update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case securityLoadedMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.analyze(msg.creds)
			m.loaded = true
		}
	}
	return nil
}

func (m *SecurityModel) analyze(creds []models.Credential) {
	m.totalCreds = len(creds)
	m.duplicates = make(map[string][]models.Credential)
	m.identities = make(map[string][]models.Credential)
	m.emptyPasswords = nil

	for _, c := range creds {
		// Track empty passwords
		if c.Password == "" {
			m.emptyPasswords = append(m.emptyPasswords, c)
			continue
		}

		// Hash password for grouping (never store plaintext in maps)
		h := sha256.Sum256([]byte(c.Password))
		hash := hex.EncodeToString(h[:])
		m.duplicates[hash] = append(m.duplicates[hash], c)

		// Group by username (skip empty)
		user := strings.TrimSpace(c.Username)
		if user != "" {
			m.identities[user] = append(m.identities[user], c)
		}
	}

	// Remove non-duplicates (only 1 credential per hash)
	for k, v := range m.duplicates {
		if len(v) <= 1 {
			delete(m.duplicates, k)
		}
	}

	// Sort keys for stable display
	m.sortedDupKeys = make([]string, 0, len(m.duplicates))
	for k := range m.duplicates {
		m.sortedDupKeys = append(m.sortedDupKeys, k)
	}
	sort.Slice(m.sortedDupKeys, func(i, j int) bool {
		return len(m.duplicates[m.sortedDupKeys[i]]) > len(m.duplicates[m.sortedDupKeys[j]])
	})

	m.sortedIdKeys = make([]string, 0, len(m.identities))
	for k := range m.identities {
		m.sortedIdKeys = append(m.sortedIdKeys, k)
	}
	sort.Slice(m.sortedIdKeys, func(i, j int) bool {
		return len(m.identities[m.sortedIdKeys[i]]) > len(m.identities[m.sortedIdKeys[j]])
	})

	m.rebuildSelectable()
}

// SelectedCredential returns the credential at the cursor, or nil.
func (m *SecurityModel) SelectedCredential() *models.Credential {
	if m.cursor >= 0 && m.cursor < len(m.selectableItems) {
		return &m.selectableItems[m.cursor]
	}
	return nil
}

func (m *SecurityModel) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "1":
		m.tab = tabOverview
		m.scroll = 0
		m.cursor = 0
		m.rebuildSelectable()
	case "2":
		m.tab = tabDuplicates
		m.scroll = 0
		m.cursor = 0
		m.rebuildSelectable()
	case "3":
		m.tab = tabIdentities
		m.scroll = 0
		m.cursor = 0
		m.rebuildSelectable()
	case "j", "down":
		if len(m.selectableItems) > 0 {
			if m.cursor < len(m.selectableItems)-1 {
				m.cursor++
			}
			m.ensureCursorInView()
		} else {
			m.scroll++
		}
	case "k", "up":
		if len(m.selectableItems) > 0 {
			if m.cursor > 0 {
				m.cursor--
			}
			m.ensureCursorInView()
		} else if m.scroll > 0 {
			m.scroll--
		}
	case "home":
		m.scroll = 0
		m.cursor = 0
	case "pgdown":
		if len(m.selectableItems) > 0 {
			m.cursor += 10
			if m.cursor >= len(m.selectableItems) {
				m.cursor = len(m.selectableItems) - 1
			}
			m.ensureCursorInView()
		} else {
			m.scroll += 10
		}
	case "pgup":
		if len(m.selectableItems) > 0 {
			m.cursor -= 10
			if m.cursor < 0 {
				m.cursor = 0
			}
			m.ensureCursorInView()
		} else {
			m.scroll -= 10
			if m.scroll < 0 {
				m.scroll = 0
			}
		}
	case "enter", "e":
		cred := m.SelectedCredential()
		if cred != nil {
			return func() tea.Msg {
				return securityEditMsg{credID: cred.ID}
			}
		}
	}
	return nil
}

// rebuildSelectable builds the flat list of selectable items for the current tab.
func (m *SecurityModel) rebuildSelectable() {
	m.selectableItems = nil
	switch m.tab {
	case tabOverview:
		// Overview: empty passwords are selectable
		m.selectableItems = append(m.selectableItems, m.emptyPasswords...)
	case tabDuplicates:
		for _, k := range m.sortedDupKeys {
			m.selectableItems = append(m.selectableItems, m.duplicates[k]...)
		}
	case tabIdentities:
		for _, k := range m.sortedIdKeys {
			m.selectableItems = append(m.selectableItems, m.identities[k]...)
		}
	}
}

// ensureCursorInView adjusts scroll so the cursor item is visible.
func (m *SecurityModel) ensureCursorInView() {
	// Each selectable item maps to roughly one line in the output.
	// This is approximate but works well enough for scrolling.
	visible := m.height - 8
	if visible < 3 {
		visible = 3
	}
	// Map cursor to approximate line position.
	// We don't have exact line mapping, so use cursor index as a heuristic.
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	}
	if m.cursor >= m.scroll+visible {
		m.scroll = m.cursor - visible + 1
	}
	if m.scroll < 0 {
		m.scroll = 0
	}
}

func (m *SecurityModel) View() string {
	if m.err != "" {
		return "\n" + styles.DangerCardStyle.Render(
			styles.DangerStyle.Render("  Error: "+m.err),
		)
	}

	if !m.loaded {
		return "\n  " + styles.MutedStyle.Render("Analyzing vault...")
	}

	var b strings.Builder

	// Title
	b.WriteString("\n")
	b.WriteString("  " + styles.TitleStyle.Render("Security Analysis"))
	b.WriteString("\n\n")

	// Tab pills
	b.WriteString("  ")
	tabs := []struct {
		key   string
		label string
		tab   securityTab
	}{
		{"1", "Overview", tabOverview},
		{"2", "Duplicate Passwords", tabDuplicates},
		{"3", "Identities", tabIdentities},
	}
	for i, t := range tabs {
		if i > 0 {
			b.WriteString(" ")
		}
		label := t.label + " (" + t.key + ")"
		if m.tab == t.tab {
			b.WriteString(styles.Badge(" "+label+" ", styles.BgDark, styles.Primary))
		} else {
			b.WriteString(styles.Badge(" "+label+" ", styles.TextDim, styles.BgCard))
		}
	}
	b.WriteString("\n\n")

	// Content
	var lines []string
	switch m.tab {
	case tabOverview:
		lines = m.renderOverview()
	case tabDuplicates:
		lines = m.renderDuplicates()
	case tabIdentities:
		lines = m.renderIdentities()
	}

	// Scrollable area
	visible := m.height - 8
	if visible < 3 {
		visible = 3
	}

	// Clamp scroll
	maxScroll := len(lines) - visible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.scroll > maxScroll {
		m.scroll = maxScroll
	}

	end := m.scroll + visible
	if end > len(lines) {
		end = len(lines)
	}

	if m.scroll > 0 {
		b.WriteString(styles.MutedStyle.Render(fmt.Sprintf("  ▲ %d more above", m.scroll)) + "\n")
	}

	for i := m.scroll; i < end; i++ {
		b.WriteString(lines[i] + "\n")
	}

	remaining := len(lines) - end
	if remaining > 0 {
		b.WriteString(styles.MutedStyle.Render(fmt.Sprintf("  ▼ %d more below", remaining)))
	}

	return b.String()
}

func (m *SecurityModel) renderOverview() []string {
	var lines []string

	// Score card
	dupCount := 0
	for _, v := range m.duplicates {
		dupCount += len(v)
	}

	scoreColor := styles.Success
	scoreLabel := "Good"
	if len(m.duplicates) > 0 {
		scoreColor = styles.Warning
		scoreLabel = "Needs Attention"
	}
	if len(m.duplicates) > 5 || len(m.emptyPasswords) > 0 {
		scoreColor = styles.Danger
		scoreLabel = "At Risk"
	}

	lines = append(lines,
		"  "+lipgloss.NewStyle().Foreground(scoreColor).Bold(true).Render("Vault Status: "+scoreLabel),
		"",
	)

	// Stats
	lines = append(lines, "  "+styles.LabelStyle.Render("Summary"))
	lines = append(lines, "")

	lines = append(lines, m.statRow("Total credentials", fmt.Sprintf("%d", m.totalCreds), styles.Text))
	lines = append(lines, m.statRow("Unique identities", fmt.Sprintf("%d", len(m.identities)), styles.Accent))
	lines = append(lines, m.statRow("Duplicate passwords", fmt.Sprintf("%d groups (%d accounts)", len(m.duplicates), dupCount), styles.Warning))
	lines = append(lines, m.statRow("Empty passwords", fmt.Sprintf("%d", len(m.emptyPasswords)), styles.Danger))
	lines = append(lines, "")

	// Empty password list
	if len(m.emptyPasswords) > 0 {
		lines = append(lines, "  "+styles.DangerStyle.Render("Accounts without passwords:"))
		for i, c := range m.emptyPasswords {
			lines = append(lines, m.renderSelectableRow(c, i))
		}
		lines = append(lines, "")
	}

	// Quick duplicate summary
	if len(m.duplicates) > 0 {
		lines = append(lines, "  "+styles.WarningStyle.Render("Password reuse detected:"))
		lines = append(lines, "  "+styles.MutedStyle.Render("Switch to tab 2 for details"))
		lines = append(lines, "")

		shown := 0
		for _, k := range m.sortedDupKeys {
			creds := m.duplicates[k]
			names := make([]string, len(creds))
			for i, c := range creds {
				names[i] = c.Name
			}
			lines = append(lines, "    "+styles.WarningStyle.Render(fmt.Sprintf("%d accounts", len(creds)))+" "+styles.MutedStyle.Render("share a password: ")+styles.DimStyle.Render(strings.Join(names, ", ")))
			shown++
			if shown >= 5 {
				remaining := len(m.duplicates) - shown
				if remaining > 0 {
					lines = append(lines, "    "+styles.MutedStyle.Render(fmt.Sprintf("... and %d more groups", remaining)))
				}
				break
			}
		}
	}

	return lines
}

func (m *SecurityModel) renderDuplicates() []string {
	var lines []string

	if len(m.duplicates) == 0 {
		lines = append(lines,
			"  "+styles.SuccessStyle.Render("No duplicate passwords found."),
			"",
			"  "+styles.MutedStyle.Render("Each credential in your vault uses a unique password."),
		)
		return lines
	}

	lines = append(lines,
		"  "+styles.WarningStyle.Render(fmt.Sprintf("%d groups of accounts share the same password", len(m.duplicates))),
		"  "+styles.MutedStyle.Render("Select an account and press enter to edit it."),
		"",
	)

	itemIdx := 0
	for i, k := range m.sortedDupKeys {
		creds := m.duplicates[k]

		lines = append(lines, "  "+styles.WarningStyle.Render(fmt.Sprintf("Group %d", i+1))+" "+styles.MutedStyle.Render(fmt.Sprintf("(%d accounts)", len(creds))))
		lines = append(lines, "  "+styles.DividerStyle.Render(strings.Repeat("─", 40)))

		for _, c := range creds {
			lines = append(lines, m.renderSelectableRow(c, itemIdx))
			itemIdx++
		}
		lines = append(lines, "")
	}

	return lines
}

func (m *SecurityModel) renderIdentities() []string {
	var lines []string

	if len(m.identities) == 0 {
		lines = append(lines,
			"  "+styles.MutedStyle.Render("No identities found."),
		)
		return lines
	}

	lines = append(lines,
		"  "+styles.AccentStyle.Render(fmt.Sprintf("%d unique identities across %d accounts", len(m.identities), m.totalCreds)),
		"  "+styles.MutedStyle.Render("Select an account and press enter to edit it."),
		"",
	)

	itemIdx := 0
	for _, user := range m.sortedIdKeys {
		creds := m.identities[user]

		lines = append(lines, "  "+styles.AccentStyle.Render(user)+" "+styles.MutedStyle.Render(fmt.Sprintf("(%d accounts)", len(creds))))
		lines = append(lines, "  "+styles.DividerStyle.Render(strings.Repeat("─", 40)))

		for _, c := range creds {
			lines = append(lines, m.renderSelectableRow(c, itemIdx))
			itemIdx++
		}
		lines = append(lines, "")
	}

	return lines
}

// renderSelectableRow renders a credential row with cursor highlight.
func (m *SecurityModel) renderSelectableRow(c models.Credential, itemIdx int) string {
	name := padRight(c.Name, 25)
	user := c.Username
	if user == "" {
		user = "-"
	}
	user = padRight(user, 20)
	url := c.URL
	if url == "" {
		url = "-"
	}

	totalW := m.width
	if totalW < 40 {
		totalW = 40
	}

	if itemIdx == m.cursor {
		row := " " + styles.SelectedStyle.Render("▸") + " " + name + " " + user + " " + url
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

	return "    " +
		styles.NormalStyle.Render(name) + " " +
		styles.DimStyle.Render(user) + " " +
		styles.MutedStyle.Render(url)
}

func (m *SecurityModel) statRow(label, value string, color lipgloss.Color) string {
	padded := padRight(label, 24)
	return "    " + styles.MutedStyle.Render(padded) + " " +
		lipgloss.NewStyle().Foreground(color).Bold(true).Render(value)
}
