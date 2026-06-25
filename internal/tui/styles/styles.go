package styles

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ── Color Palette ────────────────────────────────────────────────────────────

var (
	// Brand colors
	Primary    = lipgloss.Color("#7C3AED") // Purple 600
	PrimaryDim = lipgloss.Color("#A78BFA") // Purple 400
	Accent     = lipgloss.Color("#06B6D4") // Cyan 500
	AccentDim  = lipgloss.Color("#67E8F9") // Cyan 300

	// Semantic colors
	Success    = lipgloss.Color("#10B981") // Emerald 500
	SuccessDim = lipgloss.Color("#6EE7B7") // Emerald 300
	Warning    = lipgloss.Color("#F59E0B") // Amber 500
	WarningDim = lipgloss.Color("#FCD34D") // Amber 300
	Danger     = lipgloss.Color("#EF4444") // Red 500
	DangerDim  = lipgloss.Color("#FCA5A5") // Red 300

	// Neutrals
	Text      = lipgloss.Color("#F9FAFB") // Gray 50
	TextDim   = lipgloss.Color("#D1D5DB") // Gray 300
	Muted     = lipgloss.Color("#6B7280") // Gray 500
	Subtle    = lipgloss.Color("#374151") // Gray 700
	BgDark    = lipgloss.Color("#111827") // Gray 900
	BgCard    = lipgloss.Color("#1F2937") // Gray 800
	BgSurface = lipgloss.Color("#374151") // Gray 700
	Border    = lipgloss.Color("#4B5563") // Gray 600
)

// ── Reusable Component Styles ────────────────────────────────────────────────

var (
	// ── Typography ──

	TitleStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(Accent).
			Italic(true)

	HeadingStyle = lipgloss.NewStyle().
			Foreground(Text).
			Bold(true)

	// ── Text variants ──

	NormalStyle = lipgloss.NewStyle().
			Foreground(Text)

	DimStyle = lipgloss.NewStyle().
			Foreground(TextDim)

	MutedStyle = lipgloss.NewStyle().
			Foreground(Muted)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(Success)

	DangerStyle = lipgloss.NewStyle().
			Foreground(Danger)

	WarningStyle = lipgloss.NewStyle().
			Foreground(Warning)

	AccentStyle = lipgloss.NewStyle().
			Foreground(Accent)

	// ── Selected / cursor ──

	SelectedStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true)

	SelectedRowStyle = lipgloss.NewStyle().
				Foreground(Text).
				Background(BgSurface).
				Bold(true).
				Padding(0, 1)

	// ── Layout containers ──

	// Card is a rounded-border box for content sections.
	CardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Border).
			Padding(1, 2)

	// HighlightCard uses the primary color border.
	HighlightCardStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(Primary).
				Padding(1, 2)

	// DangerCard for destructive confirmations.
	DangerCardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Danger).
			Padding(1, 2)

	// SuccessCard for success confirmations.
	SuccessCardStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(Success).
				Padding(1, 2)

	// ── Header / Footer bars ──

	HeaderBarStyle = lipgloss.NewStyle().
			Background(BgDark).
			Foreground(Text).
			Bold(true).
			Padding(0, 2)

	FooterBarStyle = lipgloss.NewStyle().
			Foreground(Muted).
			Padding(0, 0).
			MarginTop(1)

	StatusBarStyle = lipgloss.NewStyle().
			Foreground(BgDark).
			Background(PrimaryDim).
			Bold(true).
			Padding(0, 2)

	// ── Tabs ──

	ActiveTabStyle = lipgloss.NewStyle().
			Foreground(BgDark).
			Background(Primary).
			Bold(true).
			Padding(0, 3)

	InactiveTabStyle = lipgloss.NewStyle().
				Foreground(TextDim).
				Background(BgCard).
				Padding(0, 3)

	TabGapStyle = lipgloss.NewStyle().
			Background(BgCard).
			Padding(0, 0)

	// ── Form / Inputs ──

	LabelStyle = lipgloss.NewStyle().
			Foreground(Accent).
			Bold(true)

	HintStyle = lipgloss.NewStyle().
			Foreground(Muted).
			Italic(true)

	InputGroupStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Border).
			Padding(0, 1).
			MarginBottom(1)

	InputGroupFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(Primary).
				Padding(0, 1).
				MarginBottom(1)

	// ── Badges / Pills ──

	BadgeStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Bold(true)

	// ── Divider ──

	DividerStyle = lipgloss.NewStyle().
			Foreground(Border)

	// ── Key hints ──

	KeyStyle = lipgloss.NewStyle().
			Foreground(PrimaryDim).
			Bold(true)

	KeyDescStyle = lipgloss.NewStyle().
			Foreground(Muted)

	// ── Spinner / progress ──

	SpinnerStyle = lipgloss.NewStyle().
			Foreground(Primary)

	ProgressFillStyle = lipgloss.NewStyle().
				Foreground(Primary)

	ProgressEmptyStyle = lipgloss.NewStyle().
				Foreground(Subtle)
)

// ── Helper Functions ─────────────────────────────────────────────────────────

// Divider returns a horizontal line of the given width.
func Divider(width int) string {
	return DividerStyle.Render(strings.Repeat("─", width))
}

// KeyHint renders a single key binding like "tab switch".
func KeyHint(key, desc string) string {
	return KeyStyle.Render(key) + " " + KeyDescStyle.Render(desc)
}

// HelpBar renders a row of key hints separated by dots.
func HelpBar(hints ...string) string {
	return FooterBarStyle.Render("  " + strings.Join(hints, "  "+MutedStyle.Render("·")+"  "))
}

// Badge renders a small colored pill with text.
func Badge(text string, fg, bg lipgloss.Color) string {
	return BadgeStyle.
		Foreground(fg).
		Background(bg).
		Render(text)
}

// EnvBadge returns a styled environment badge.
func EnvBadge(env string) string {
	switch env {
	case "production":
		return Badge(" PROD ", Text, Danger)
	case "staging":
		return Badge(" STG ", BgDark, Warning)
	case "development":
		return Badge(" DEV ", BgDark, Success)
	default:
		return Badge(" "+strings.ToUpper(env)+" ", Text, Muted)
	}
}

// ProgressDots renders step progress like "● ● ○ ○".
func ProgressDots(current, total int) string {
	var b strings.Builder
	for i := 0; i < total; i++ {
		if i > 0 {
			b.WriteString(" ")
		}
		if i < current {
			b.WriteString(ProgressFillStyle.Render("●"))
		} else if i == current {
			b.WriteString(SpinnerStyle.Render("◉"))
		} else {
			b.WriteString(ProgressEmptyStyle.Render("○"))
		}
	}
	return b.String()
}

// StrengthMeter renders a password strength bar.
func StrengthMeter(length int, width int) string {
	// Map password length to a 0-100 strength score.
	score := length * 4
	if score > 100 {
		score = 100
	}
	filled := score * width / 100
	if filled < 1 && score > 0 {
		filled = 1
	}

	var color lipgloss.Color
	var label string
	switch {
	case score < 30:
		color = Danger
		label = "Weak"
	case score < 60:
		color = Warning
		label = "Fair"
	case score < 80:
		color = AccentDim
		label = "Good"
	default:
		color = Success
		label = "Strong"
	}

	bar := lipgloss.NewStyle().Foreground(color).Render(strings.Repeat("█", filled)) +
		ProgressEmptyStyle.Render(strings.Repeat("░", width-filled))

	return bar + "  " + lipgloss.NewStyle().Foreground(color).Bold(true).Render(label)
}

// Logo returns the ASCII art logo for the splash screen.
func Logo() string {
	logo := `
   ███████╗███████╗ ██████╗██████╗ ███████╗████████╗███████╗
   ██╔════╝██╔════╝██╔════╝██╔══██╗██╔════╝╚══██╔══╝██╔════╝
   ███████╗█████╗  ██║     ██████╔╝█████╗     ██║   ███████╗
   ╚════██║██╔══╝  ██║     ██╔══██╗██╔══╝     ██║   ╚════██║
   ███████║███████╗╚██████╗██║  ██║███████╗   ██║   ███████║
   ╚══════╝╚══════╝ ╚═════╝╚═╝  ╚═╝╚══════╝   ╚═╝   ╚══════╝`

	return lipgloss.NewStyle().Foreground(Primary).Render(logo)
}

// LogoSmall returns a compact logo for the header bar.
func LogoSmall() string {
	return lipgloss.NewStyle().
		Foreground(Primary).
		Bold(true).
		Render("◆ SECRETS")
}
