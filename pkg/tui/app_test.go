package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/theshedman/shedman/pkg/tui/views"
)

func TestAppModel_Update_StateTransition(t *testing.T) {
	// Initialize App
	app := newAppModel(nil, nil)

	// Assert initial state is Dashboard
	assert.Equal(t, viewDashboard, app.active, "Initial active view should be Dashboard")

	// Trigger Tab to switch focus to content
	msg := tea.KeyMsg{Type: tea.KeyTab}
	m, _ := app.Update(msg)
	newApp := m.(appModel)
	assert.Equal(t, 1, newApp.focus, "Focus should switch to Content")
}

func TestSidebar_Items(t *testing.T) {
	app := newAppModel(nil, nil)
	items := app.sidebar.Items()

	assert.NotEmpty(t, items, "Sidebar should have items")

	// Check first item type
	_, ok := items[0].(dashboardItem)
	assert.True(t, ok, "Items should be of type dashboardItem")
}

func TestSearchModel_Update(t *testing.T) {
	// Initialize
	m := views.NewSearchModel(nil)

	// Test Input
	// Send 'a'
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	model := newM.(views.SearchModel)

	assert.Equal(t, "a", model.TextInput.Value(), "Text input should update")

	// Test Esc from Search (Should clear or Focus Sidebar)
	// Case 1: Input not empty (Clear)
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	mClean, _ := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.Equal(t, "", mClean.(views.SearchModel).TextInput.Value(), "Esc should clear input")

	// Case 2: Input empty (Focus Sidebar)
	mEmpty := views.NewSearchModel(nil)
	_, cmd := mEmpty.Update(tea.KeyMsg{Type: tea.KeyEsc})

	// Check if cmd returns FocusSidebarMsg
	assert.NotNil(t, cmd, "Esc on empty input should return a command (FocusSidebarMsg)")

	// Execute command to verify type
	if cmd != nil {
		msg := cmd()
		_, ok := msg.(views.FocusSidebarMsg)
		assert.True(t, ok, "Command should return FocusSidebarMsg")
	}
}
