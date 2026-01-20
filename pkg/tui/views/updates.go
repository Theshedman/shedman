package views

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/tui/theme"
)

type UpdatesModel struct {
	List    list.Model
	Spinner spinner.Model
	core    *core.Engine
	loading bool
	err     error
	diffs   []core.PackageDiff
}

func NewUpdatesModel(c *core.Engine) UpdatesModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = theme.HighlightStyle

	// Empty list initially
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Pending Updates"
	l.SetShowHelp(false)
	l.Styles.Title = theme.TitleStyle

	return UpdatesModel{
		List:    l,
		Spinner: s,
		core:    c,
		loading: true,
	}
}

func (m UpdatesModel) Init() tea.Cmd {
	m.loading = true
	return tea.Batch(m.Spinner.Tick, fetchUpdatesCmd(m.core))
}

// Custom message type to ensure routing
type RequestUpdateMsg struct{}

// Custom message type to ensure routing
type UpdatesFinishedMsg struct {
	Diffs []core.PackageDiff
	Err   error
}

// Command to fetch updates
func fetchUpdatesCmd(eng *core.Engine) tea.Cmd {
	return func() tea.Msg {
		diffs, err := eng.Diff()
		return UpdatesFinishedMsg{Diffs: diffs, Err: err}
	}
}

func (m UpdatesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			// Trigger update request
			return m, func() tea.Msg { return RequestUpdateMsg{} }
		}

	case tea.WindowSizeMsg:
		m.List.SetSize(msg.Width, msg.Height)

	case spinner.TickMsg:
		if m.loading {
			var sCmd tea.Cmd
			m.Spinner, sCmd = m.Spinner.Update(msg)
			cmds = append(cmds, sCmd)
		}

	case UpdatesFinishedMsg:
		m.loading = false
		if msg.Err != nil {
			m.err = msg.Err
			m.List.Title = fmt.Sprintf("Error: %s", msg.Err.Error())
		} else {
			m.diffs = msg.Diffs
			items := make([]list.Item, len(msg.Diffs))
			for i, d := range msg.Diffs {
				items[i] = updateItem{diff: d}
			}
			m.List.SetItems(items)
			if len(items) == 0 {
				m.List.Title = "System Up to Date"
			} else {
				m.List.Title = fmt.Sprintf("Pending Updates (%d)", len(items))
			}
		}

	case string:
		if msg == "reload_updates" {
			return m, m.LoadTriggerCmd()
		}
	}

	if !m.loading {
		m.List, cmd = m.List.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m UpdatesModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n  %s\n", theme.ErrorStyle.Render("Error: "+m.err.Error()))
	}
	if m.loading {
		return fmt.Sprintf("\n  %s Checking for updates...\n\n", m.Spinner.View())
	}
	return m.List.View()
}

// LoadTriggerCmd is a helper to start loading updates
func (m *UpdatesModel) LoadTriggerCmd() tea.Cmd {
	m.loading = true
	return tea.Batch(m.Spinner.Tick, fetchUpdatesCmd(m.core))
}

// -- List Item --

type updateItem struct {
	diff core.PackageDiff
}

func (i updateItem) Title() string {
	return fmt.Sprintf("%s %s -> %s", i.diff.Name, i.diff.OldVersion, i.diff.NewVersion)
}

func (i updateItem) Description() string {
	s := fmt.Sprintf("Size: %s", formatSize(i.diff.SizeDelta))
	if len(i.diff.CVEs) > 0 {
		s += fmt.Sprintf(" | %d SECURITY ISSUES", len(i.diff.CVEs))
	}
	if i.diff.Pacnew {
		s += " | PACNEW"
	}
	return s
}

func (i updateItem) FilterValue() string { return i.diff.Name }

// Helper for formatting size
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
