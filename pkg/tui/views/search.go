package views

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/tui/theme"
)

// Messages
type FocusSidebarMsg struct{}

const (
	focusInput = iota
	focusList
)

type installFinishedMsg struct {
	pkg string
}

type RequestInstallMsg struct {
	Package string
}

type SearchModel struct {
	TextInput textinput.Model
	Results   list.Model
	core      *core.Engine
	searching bool
	err       error
	focus     int
}

func NewSearchModel(c *core.Engine) SearchModel {
	ti := textinput.New()
	ti.Placeholder = "Type package name..."
	ti.Focus()
	ti.CharLimit = 156
	// Width is managed by update/layout logic now

	// Results list
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Search Results"
	l.SetShowHelp(false)
	l.Styles.Title = theme.TitleStyle

	return SearchModel{
		TextInput: ti,
		Results:   l,
		core:      c,
	}
}

func (m SearchModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m SearchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			if m.focus == focusList {
				m.focus = focusInput
				return m, nil
			}
			if m.TextInput.Value() != "" {
				m.TextInput.SetValue("")
			} else {
				// Return focus to sidebar
				return m, func() tea.Msg { return FocusSidebarMsg{} }
			}
		case tea.KeyEnter:
			switch m.focus {
			case focusInput:
				query := m.TextInput.Value()
				if query != "" {
					m.searching = true
					return m, func() tea.Msg {
						res, err := m.core.Search(query)
						if err != nil {
							return err
						}
						return res
					}
				}
			case focusList:
				// Install selected package
				item, ok := m.Results.SelectedItem().(searchResultItem)
				if ok {
					// Request embedded install via app model negotiation
					return m, func() tea.Msg {
						return RequestInstallMsg{Package: item.pkg.Name}
					}
				}
			}
		case tea.KeyDown:
			if m.focus == focusInput && len(m.Results.Items()) > 0 {
				m.focus = focusList
				m.TextInput.Blur()
				return m, nil
			}
		case tea.KeyUp:
			if m.focus == focusList && m.Results.Index() == 0 {
				m.focus = focusInput
				m.TextInput.Focus()
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		headerHeight := 6
		m.Results.SetSize(msg.Width, msg.Height-headerHeight)
		m.TextInput.Width = msg.Width

	// Handle Search Results
	case []core.PackageInfo:
		m.searching = false
		items := make([]list.Item, len(msg))
		for i, p := range msg {
			items[i] = searchResultItem{pkg: p}
		}
		m.Results.SetItems(items)

	case error:
		m.searching = false
		m.err = msg
		m.Results.Title = fmt.Sprintf("Error: %s", msg.Error())

	case installFinishedMsg:
		m.Results.Title = fmt.Sprintf("Installed: %s", msg.pkg)
	}

	if m.focus == focusInput {
		m.TextInput, cmd = m.TextInput.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		m.Results, cmd = m.Results.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m SearchModel) View() string {
	return fmt.Sprintf(
		"Search Packages\n\n%s\n\n%s\n\n%s",
		m.TextInput.View(),
		theme.SubtextStyle.Render("Press [Enter] to install selected package"),
		m.Results.View(),
	)
}

// searchResultItem implements list.Item
type searchResultItem struct {
	pkg core.PackageInfo
}

func (i searchResultItem) Title() string { return i.pkg.Name }
func (i searchResultItem) Description() string {
	return fmt.Sprintf("%s (%s)", i.pkg.Version, i.pkg.Source)
}
func (i searchResultItem) FilterValue() string { return i.pkg.Name }
