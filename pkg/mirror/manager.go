package mirror

import (
	"fmt"
)

// Manager handles mirror operations
type Manager struct {
	backend MirrorBackend
}

// New creates a new mirror manager with reflector backend
func New() *Manager {
	return &Manager{
		backend: NewReflectorBackend(),
	}
}

// NewWithBackend creates a manager with a specific backend
func NewWithBackend(b MirrorBackend) *Manager {
	return &Manager{
		backend: b,
	}
}

// List lists configured mirrors
func (m *Manager) List() ([]Mirror, error) {
	if m.backend == nil {
		return nil, fmt.Errorf("no mirror backend configured")
	}
	return m.backend.List()
}

// Test tests mirror speeds
func (m *Manager) Test() ([]Mirror, error) {
	if m.backend == nil {
		return nil, fmt.Errorf("no mirror backend configured")
	}
	return m.backend.Test()
}

// Select select top N fastest mirrors
func (m *Manager) Select(topN int, countries []string, sort string) error {
	if m.backend == nil {
		return fmt.Errorf("no mirror backend configured")
	}
	return m.backend.Select(topN, countries, sort)
}
