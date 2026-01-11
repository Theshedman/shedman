package mirror

import "time"

// Manager handles mirror operations
type Manager struct {
	// config path
}

// New creates a new mirror manager
func New() *Manager {
	return &Manager{}
}

// Mirror represents a package mirror
type Mirror struct {
	URL     string
	Country string
	Speed   time.Duration
}

// List lists configured mirrors
func (m *Manager) List() ([]Mirror, error) {
	return nil, nil
}

// Test tests mirror speeds
func (m *Manager) Test() ([]Mirror, error) {
	return nil, nil
}

// Select select top N fastest mirrors
func (m *Manager) Select(topN int) error {
	return nil
}
