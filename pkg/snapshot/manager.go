package snapshot

import "fmt"

// Manager handles snapshot operations
type Manager struct {
	backend SnapshotBackend
}

// New creates a new snapshot manager with a default backend (to be implemented)
func New() *Manager {
	return &Manager{}
}

// NewWithBackend creates a manager with a specific backend
func NewWithBackend(b SnapshotBackend) *Manager {
	return &Manager{
		backend: b,
	}
}

// Create creates a new system snapshot
func (m *Manager) Create(opts CreateOptions) (*Snapshot, error) {
	if m.backend == nil {
		return nil, fmt.Errorf("no snapshot backend configured")
	}
	return m.backend.Create(opts)
}

// Restore restores a snapshot
func (m *Manager) Restore(id string, opts RestoreOptions) error {
	if m.backend == nil {
		return fmt.Errorf("no snapshot backend configured")
	}
	return m.backend.Restore(id, opts)
}

// List returns a list of snapshots
func (m *Manager) List() ([]*Snapshot, error) {
	if m.backend == nil {
		return nil, fmt.Errorf("no snapshot backend configured")
	}
	// Return list of pointers to match interface
	return m.backend.List()
}
