package svc

// Manager handles service operations
type Manager struct {
	// systemd connection?
}

// New creates a new service manager
func New() *Manager {
	return &Manager{}
}

// Service represents a system service
type Service struct {
	Name    string
	Active  bool
	Enabled bool
}

// List lists services
func (m *Manager) List() ([]Service, error) {
	return nil, nil
}

// Enable enables a service
func (m *Manager) Enable(name string) error {
	return nil
}

// Start starts a service
func (m *Manager) Start(name string) error {
	return nil
}

// Status gets service status
func (m *Manager) Status(name string) (*Service, error) {
	return nil, nil
}
