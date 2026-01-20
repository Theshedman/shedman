package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/theshedman/shedman/pkg/config"
)

func TestConflictModel_Update(t *testing.T) {
	initial := newConflictModel("test.conf", "diff content")

	tests := []struct {
		name     string
		key      string
		expected config.Action
	}{
		{"Keep User (k)", "k", config.ActionKeepUser},
		{"Keep User (K)", "K", config.ActionKeepUser},
		{"Update (u)", "u", config.ActionUpdate},
		{"Update (U)", "U", config.ActionUpdate},
		{"Reset (r)", "r", config.ActionReset},
		{"Reset (R)", "R", config.ActionReset},
		{"Quit (q)", "q", config.ActionKeepUser},
		{"Quit (esc)", "esc", config.ActionKeepUser},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Trigger Update with KeyMsg
			model, cmd := initial.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)})

			// For special keys like esc
			if tt.key == "esc" {
				model, cmd = initial.Update(tea.KeyMsg{Type: tea.KeyEsc})
			}

			// Validate command is Quit
			assert.NotNil(t, cmd, "Expected a command returned (Quit)")

			// Validate state
			m := model.(conflictModel)
			assert.True(t, m.quitting, "Expected quitting to be true")
			assert.Equal(t, tt.expected, m.selectedAction, "Action mismatch")
		})
	}
}

func TestConflictModel_Resize(t *testing.T) {
	initial := newConflictModel("test.conf", "long diff content")

	// Send WindowSizeMsg
	width, height := 100, 50
	model, _ := initial.Update(tea.WindowSizeMsg{Width: width, Height: height})

	m := model.(conflictModel)

	// Assert viewport dimensions are updated (this assumes we add a viewport field)
	// We expect the viewport height to be less than window height (accounting for header/footer)
	assert.Equal(t, width, m.viewport.Width, "Viewport width mismatch")
	assert.Less(t, m.viewport.Height, height, "Viewport height should account for UI chrome")
	assert.Greater(t, m.viewport.Height, 0, "Viewport height should be positive")
}
