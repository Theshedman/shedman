package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/theshedman/shedman/pkg/tui/theme"
)

// dashboardItem represents a menu item in the sidebar
type dashboardItem struct {
	title, desc string
	view        sessionState
}

func (i dashboardItem) Title() string       { return i.title }
func (i dashboardItem) Description() string { return i.desc }
func (i dashboardItem) FilterValue() string { return i.title }

// dashboardDelegate handles customized rendering for the sidebar
type dashboardDelegate struct{}

func (d dashboardDelegate) Height() int                             { return 1 }
func (d dashboardDelegate) Spacing() int                            { return 0 }
func (d dashboardDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d dashboardDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(dashboardItem)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i.title)

	// Use Centralized Styles
	fn := theme.ItemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return theme.SelectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	_, _ = fmt.Fprint(w, fn(str))
}
