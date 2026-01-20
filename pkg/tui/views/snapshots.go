package views

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/snapshot"
	"github.com/theshedman/shedman/pkg/tui/theme"
)

type SnapshotsModel struct {
	List    list.Model
	Spinner spinner.Model
	core    *core.Engine
	loading bool
	err     error
	items   []snapshot.Snapshot
}

func NewSnapshotsModel(c *core.Engine) SnapshotsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = theme.HighlightStyle

	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "System Snapshots"
	l.SetShowHelp(false)
	l.Styles.Title = theme.TitleStyle

	return SnapshotsModel{
		List:    l,
		Spinner: s,
		core:    c,
		loading: true,
	}
}

func (m SnapshotsModel) Init() tea.Cmd {
	m.loading = true
	return tea.Batch(m.Spinner.Tick, fetchSnapshotsCmd(m.core))
}

type SnapshotsFinishedMsg struct {
	Snapshots []snapshot.Snapshot
	Err       error
}

func fetchSnapshotsCmd(eng *core.Engine) tea.Cmd {
	return func() tea.Msg {
		sm := eng.GetSnapshotManager()
		if sm == nil {
			return SnapshotsFinishedMsg{Err: fmt.Errorf("snapshot manager not available")}
		}
		// List Local Snapshots
		snaps, err := sm.List(snapshot.ListOptions{Remote: false})
		return SnapshotsFinishedMsg{Snapshots: snaps, Err: err}
	}
}

func (m SnapshotsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.List.SetSize(msg.Width, msg.Height)

	case spinner.TickMsg:
		if m.loading {
			var sCmd tea.Cmd
			m.Spinner, sCmd = m.Spinner.Update(msg)
			cmds = append(cmds, sCmd)
		}

	case SnapshotsFinishedMsg:
		m.loading = false
		if msg.Err != nil {
			m.err = msg.Err
			m.List.Title = fmt.Sprintf("Error: %s", msg.Err.Error())
		} else {
			m.items = msg.Snapshots
			items := make([]list.Item, len(msg.Snapshots))
			for i, s := range msg.Snapshots {
				items[i] = snapshotItem{s: s}
			}
			m.List.SetItems(items)
			m.List.Title = fmt.Sprintf("System Snapshots (%d)", len(items))
		}

	case tea.KeyMsg:
		if !m.loading {
			switch msg.String() {
			case "c":
				return m, func() tea.Msg { return RequestSnapshotCreateMsg{} }
			case "p":
				item, ok := m.List.SelectedItem().(snapshotItem)
				if ok {
					return m, func() tea.Msg { return RequestSnapshotPushMsg{ID: item.s.ID} }
				}
			case "enter":
				item, ok := m.List.SelectedItem().(snapshotItem)
				if ok {
					return m, func() tea.Msg { return RequestSnapshotRestoreMsg{ID: item.s.ID} }
				}
			}
		}
	}

	if !m.loading {
		m.List, cmd = m.List.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

type RequestSnapshotCreateMsg struct {
	Description string
}

type RequestSnapshotPushMsg struct {
	ID string
}

type RequestSnapshotRestoreMsg struct {
	ID string
}

func (m SnapshotsModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n  %s\n", theme.ErrorStyle.Render("Error: "+m.err.Error()))
	}
	if m.loading {
		return fmt.Sprintf("\n  %s Loading snapshots...\n\n", m.Spinner.View())
	}
	// Add help text
	help := "\n  [c] Create  [p] Push to Remote  [Enter] Restore\n"
	return m.List.View() + theme.SubtextStyle.Render(help)
}

// LoadTriggerCmd triggers snapshot loading
func (m *SnapshotsModel) LoadTriggerCmd() tea.Cmd {
	m.loading = true
	return tea.Batch(m.Spinner.Tick, fetchSnapshotsCmd(m.core))
}

// -- List Item --

type snapshotItem struct {
	s snapshot.Snapshot
}

func (i snapshotItem) Title() string {
	return fmt.Sprintf("#%s %s", i.s.ID, i.s.Timestamp.Format("2006-01-02 15:04"))
}

func (i snapshotItem) Description() string {
	desc := i.s.Description
	if desc == "" {
		desc = i.s.Type
	}
	return fmt.Sprintf("%s | %s", desc, i.s.Type)
}

func (i snapshotItem) FilterValue() string { return i.s.ID + i.s.Description }
