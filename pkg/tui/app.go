package tui

import "github.com/theshedman/shedman/pkg/core"

// App represents the TUI application
type App struct {
	core *core.Engine
}

// New creates a new TUI app
func New(c *core.Engine) *App {
	return &App{
		core: c,
	}
}

// Run runs the TUI
func (a *App) Run() error {
	return nil
}
