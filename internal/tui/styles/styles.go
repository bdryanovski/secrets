package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	Primary   = lipgloss.Color("#7C3AED") // Purple
	Secondary = lipgloss.Color("#06B6D4") // Cyan
	Success   = lipgloss.Color("#10B981") // Green
	Warning   = lipgloss.Color("#F59E0B") // Amber
	Danger    = lipgloss.Color("#EF4444") // Red
	Muted     = lipgloss.Color("#6B7280") // Gray
	Text      = lipgloss.Color("#F9FAFB") // Light
	BgDark    = lipgloss.Color("#1F2937") // Dark bg

	// Title style
	TitleStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true).
			Padding(0, 1)

	// Subtitle style
	SubtitleStyle = lipgloss.NewStyle().
			Foreground(Secondary).
			Italic(true)

	// Selected item style
	SelectedStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true)

	// Normal item style
	NormalStyle = lipgloss.NewStyle().
			Foreground(Text)

	// Muted text style
	MutedStyle = lipgloss.NewStyle().
			Foreground(Muted)

	// Success text style
	SuccessStyle = lipgloss.NewStyle().
			Foreground(Success)

	// Danger text style
	DangerStyle = lipgloss.NewStyle().
			Foreground(Danger)

	// Warning text style
	WarningStyle = lipgloss.NewStyle().
			Foreground(Warning)

	// StatusBar style
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(Text).
			Background(BgDark).
			Padding(0, 1)

	// Help style
	HelpStyle = lipgloss.NewStyle().
			Foreground(Muted).
			Padding(1, 0)

	// Box style for detail views
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Primary).
			Padding(1, 2)

	// Input label style
	LabelStyle = lipgloss.NewStyle().
			Foreground(Secondary).
			Bold(true)

	// Tab styles
	ActiveTabStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true).
			Underline(true).
			Padding(0, 2)

	InactiveTabStyle = lipgloss.NewStyle().
				Foreground(Muted).
				Padding(0, 2)
)
