package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/theshedman/shedman/pkg/config"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	diffStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(0, 1).
			Width(80)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000"))
)

// TUIConflictResolver implements config.ConflictResolver using Bubbletea
type TUIConflictResolver struct{}

// NewConflictResolver creates a new interactive resolver
func NewConflictResolver() *TUIConflictResolver {
	return &TUIConflictResolver{}
}

// Resolve starts the TUI to resolve a conflict
func (r *TUIConflictResolver) Resolve(file string, diff string) (config.Action, error) {
	initialModel := newConflictModel(file, diff)
	p := tea.NewProgram(initialModel)

	// Run the program
	finalModel, err := p.Run()
	if err != nil {
		return config.ActionKeepUser, err
	}

	// Cast back to our model to get the result
	if m, ok := finalModel.(conflictModel); ok {
		if m.err != nil {
			return config.ActionKeepUser, m.err
		}
		return m.selectedAction, nil
	}

	return config.ActionKeepUser, fmt.Errorf("tui model type assertion failed")
}

// conflictModel holds the state of the TUI
type conflictModel struct {
	file           string
	diff           string
	selectedAction config.Action
	quitting       bool
	err            error
	viewportHeight int
	viewportWidth  int
}

func newConflictModel(file, diff string) conflictModel {
	return conflictModel{
		file:           file,
		diff:           diff,
		selectedAction: config.ActionKeepUser, // Default if quit
	}
}

func (m conflictModel) Init() tea.Cmd {
	return nil
}

func (m conflictModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			// Default to keeping user on abort
			m.selectedAction = config.ActionKeepUser
			return m, tea.Quit

		case "k", "K":
			m.selectedAction = config.ActionKeepUser
			m.quitting = true
			return m, tea.Quit

		case "u", "U":
			m.selectedAction = config.ActionUpdate
			m.quitting = true
			return m, tea.Quit

		case "r", "R":
			m.selectedAction = config.ActionReset
			m.quitting = true
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.viewportWidth = msg.Width
		m.viewportHeight = msg.Height
	}

	return m, nil
}

func (m conflictModel) View() string {
	if m.quitting {
		return ""
	}

	s := strings.Builder{}

	// Header
	s.WriteString(titleStyle.Render("Configuration Conflict"))
	s.WriteString(fmt.Sprintf("\n\nFile: %s\n", m.file))

	// Diff View (Truncated or scrollable in future)
	// For now, simple rendering
	diffView := diffStyle.Render(m.diff)
	s.WriteString(diffView)
	s.WriteString("\n\n")

	// Options
	s.WriteString("Resolution Options:\n")
	s.WriteString("  [K] Keep Your Version (Default)\n")
	s.WriteString("  [U] Update to Package Version (Backs up valid user config)\n")
	s.WriteString("  [R] Reset/Overwrite (Same as Update but explicit)\n")
	s.WriteString("  [Q] Quit (Keeps user version)\n")

	return s.String()
}
