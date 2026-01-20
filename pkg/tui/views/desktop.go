package views

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/de"
	"github.com/theshedman/shedman/pkg/tui/theme"
)

type DesktopModel struct {
	List  list.Model
	core  *core.Engine
	deMgr *de.Manager
	err   error
}

func NewDesktopModel(c *core.Engine, dm *de.Manager) DesktopModel {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Desktop Environments"
	l.SetShowHelp(false)
	l.Styles.Title = theme.TitleStyle

	return DesktopModel{
		List:  l,
		core:  c,
		deMgr: dm,
	}
}

type DesktopListFinishedMsg struct {
	Items []de.DesktopEnvironment
	Err   error
}

func (m DesktopModel) Init() tea.Cmd {
	return func() tea.Msg {
		if m.deMgr == nil {
			return DesktopListFinishedMsg{Err: fmt.Errorf("DE manager not available")}
		}
		des, err := m.deMgr.List()
		return DesktopListFinishedMsg{Items: des, Err: err}
	}
}

func (m DesktopModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.List.SetSize(msg.Width, msg.Height)

	case DesktopListFinishedMsg:
		if msg.Err != nil {
			m.err = msg.Err
			m.List.Title = fmt.Sprintf("Error: %s", msg.Err.Error())
		} else {
			items := make([]list.Item, len(msg.Items))
			for i, de := range msg.Items {
				items[i] = deItem{de: de}
			}
			m.List.SetItems(items)
		}

	case error: // Fallback
		m.err = msg
		m.List.Title = fmt.Sprintf("Error: %s", msg.Error())
	}

	m.List, cmd = m.List.Update(msg)
	return m, cmd
}

func (m DesktopModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n  %s\n", theme.ErrorStyle.Render("Error: "+m.err.Error()))
	}
	return m.List.View()
}

type deItem struct {
	de de.DesktopEnvironment
}

func (i deItem) Title() string {
	status := ""
	if i.de.Installed {
		status = " [Installed]"
	}
	return i.de.Name + status
}

func (i deItem) Description() string {
	return fmt.Sprintf("Service: %s | Group: %s", i.de.Service, i.de.Group)
}

func (i deItem) FilterValue() string { return i.de.Name }
