package de

import "github.com/theshedman/shedman/pkg/core"

// Manager handles desktop environment operations
type Manager struct {
	core *core.Engine
}

// New creates a new DE manager
func New(c *core.Engine) *Manager {
	return &Manager{
		core: c,
	}
}

// DesktopEnvironment represents a DE
type DesktopEnvironment struct {
	Name string
}

// Switch switches to the specified DE
func (m *Manager) Switch(name string) error {
	return nil
}

// List lists available DEs
func (m *Manager) List() ([]DesktopEnvironment, error) {
	return nil, nil
}
