package keyring

// Manager handles GPG keyring operations
type Manager struct {
	path string
}

// New creates a new keyring manager
func New(path string) *Manager {
	return &Manager{
		path: path,
	}
}

// Key represents a GPG key
type Key struct {
	ID    string
	Name  string
	Email string
}

// List lists keys
func (m *Manager) List() ([]Key, error) {
	return nil, nil
}

// Add adds a key
func (m *Manager) Add(id string) error {
	return nil
}

// Remove removes a key
func (m *Manager) Remove(id string) error {
	return nil
}
