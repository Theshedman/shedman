package theme

import "github.com/theshedman/shedman/pkg/core"

// Manager handles theme operations
type Manager struct {
	core *core.Engine
}

// New creates a new theme manager
func New(c *core.Engine) *Manager {
	return &Manager{
		core: c,
	}
}

// Theme represents a system theme
type Theme struct {
	Name string
}

// Apply applies the specified theme
func (m *Manager) Apply(name string) error {
	return nil
}

// List lists available themes
func (m *Manager) List() ([]Theme, error) {
	return nil, nil
}
