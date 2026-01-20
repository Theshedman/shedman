package theme

import "github.com/charmbracelet/lipgloss"

// Theme Colors (Catppuccin Mocha Inspired)
const (
	ColorBackground = "#1E1E2E"
	ColorForeground = "#CDD6F4"
	ColorBorder     = "#585B70"
	ColorActive     = "#CBA6F7" // Mauve
	ColorHighlight  = "#89B4FA" // Blue
	ColorRed        = "#F38BA8"
	ColorGreen      = "#A6E3A1"
	ColorYellow     = "#F9E2AF"
	ColorSubtext    = "#6C7086"
)

var (
	// Border Styles
	DocStyle = lipgloss.NewStyle().Margin(1, 2)

	ActiveBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(ColorActive)).
				Padding(0, 1)

	InactiveBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(ColorBorder)).
				Padding(0, 1)

	// Text Styles
	TitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorBackground)).
			Background(lipgloss.Color(ColorActive)).
			Padding(0, 1).
			Bold(true)

	HighlightStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorHighlight))
	ErrorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorRed))
	SuccessStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGreen))
	SubtextStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorSubtext))

	// List Styles
	ItemStyle         = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color(ColorForeground))
	SelectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(0).
				Foreground(lipgloss.Color(ColorActive)).
				Bold(true).
				SetString("> ")

	// Layout Styles
	ContentStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Foreground(lipgloss.Color(ColorForeground))

	DialogStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorActive)).
			Padding(1, 2).
			Align(lipgloss.Center)
)
