package config

import "github.com/theshedman/shedman/pkg/core"

// Manager handles configuration package operations
type Manager struct {
	core *core.Engine
}

// New creates a new config manager
func New(c *core.Engine) *Manager {
	return &Manager{
		core: c,
	}
}

// ConfigPackage represents a configuration package
type ConfigPackage struct {
	Name        string
	Description string
	Version     string
}

// Install installs a configuration package
func (m *Manager) Install(name string) error {
	return nil
}

// List returns available configuration packages
func (m *Manager) List() ([]ConfigPackage, error) {
	return nil, nil
}
