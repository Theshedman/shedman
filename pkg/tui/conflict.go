package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/theshedman/shedman/pkg/config"
)

var (
	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1)
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
	p := tea.NewProgram(initialModel, tea.WithAltScreen())

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
	viewport       viewport.Model
	ready          bool
}

func newConflictModel(file, diff string) conflictModel {
	vp := viewport.New(0, 0)
	vp.SetContent(ColorizeDiff(diff))

	return conflictModel{
		file:           file,
		diff:           diff,
		selectedAction: config.ActionKeepUser,
		viewport:       vp,
	}
}

func (m conflictModel) Init() tea.Cmd {
	return nil
}

func (m conflictModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			m.selectedAction = config.ActionKeepUser
			return m, tea.Quit

		case "k", "K":
			// Intercept K for Keep (otherwise viewport uses k for up)
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
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.SetContent(ColorizeDiff(m.diff))
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}
	}

	// Handle viewport updates
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m conflictModel) View() string {
	if m.quitting {
		return ""
	}
	if !m.ready {
		return "\n  Initializing..."
	}

	return fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView())
}

func (m conflictModel) headerView() string {
	s := strings.Builder{}
	s.WriteString(titleStyle.Render("Configuration Conflict"))
	s.WriteString(fmt.Sprintf("\n\nFile: %s\n", m.file))
	return s.String()
}

func (m conflictModel) footerView() string {
	s := strings.Builder{}
	s.WriteString("\nResolution Options:\n")
	s.WriteString("  [K] Keep Your Version (Default)\n")
	s.WriteString("  [U] Update to Package Version (Backs up valid user config)\n")
	s.WriteString("  [R] Reset/Overwrite (Same as Update but explicit)\n")
	s.WriteString("  [Q] Quit (Keeps user version)\n")
	s.WriteString("  [↑/↓] Scroll Diff")
	return s.String()
}

// ColorizeDiff colors diff output
func ColorizeDiff(diff string) string {
	lines := strings.Split(diff, "\n")
	var out []string
	for _, line := range lines {
		if strings.HasPrefix(line, "+") {
			out = append(out, lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render(line)) // Green
		} else if strings.HasPrefix(line, "-") {
			out = append(out, lipgloss.NewStyle().Foreground(lipgloss.Color("160")).Render(line)) // Red
		} else if strings.HasPrefix(line, "@@") {
			out = append(out, lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Render(line)) // Grey for metadata
		} else {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}
