package svc

import "fmt"

// Manager handles service operations
type Manager struct {
	backend ServiceBackend
}

// New creates a new service manager with systemd backend
func New() *Manager {
	return &Manager{
		backend: NewSystemdBackend(),
	}
}

// NewWithBackend creates a manager with a specific backend
func NewWithBackend(b ServiceBackend) *Manager {
	return &Manager{
		backend: b,
	}
}

// Service represents a system service
type Service struct {
	Name    string
	Active  bool
	Enabled bool
}

// List lists services
func (m *Manager) List() ([]Service, error) {
	if m.backend == nil {
		return nil, fmt.Errorf("no service backend configured")
	}
	return m.backend.List()
}

// Enable enables a service
func (m *Manager) Enable(name string) error {
	if m.backend == nil {
		return fmt.Errorf("no service backend configured")
	}
	return m.backend.Enable(name)
}

// Disable disables a service
func (m *Manager) Disable(name string) error {
	if m.backend == nil {
		return fmt.Errorf("no service backend configured")
	}
	return m.backend.Disable(name)
}

// Start starts a service
func (m *Manager) Start(name string) error {
	if m.backend == nil {
		return fmt.Errorf("no service backend configured")
	}
	return m.backend.Start(name)
}

// Stop stops a service
func (m *Manager) Stop(name string) error {
	if m.backend == nil {
		return fmt.Errorf("no service backend configured")
	}
	return m.backend.Stop(name)
}

// Restart restarts a service
func (m *Manager) Restart(name string) error {
	if m.backend == nil {
		return fmt.Errorf("no service backend configured")
	}
	return m.backend.Restart(name)
}

// Status gets service status
func (m *Manager) Status(name string) (*Service, error) {
	if m.backend == nil {
		return nil, fmt.Errorf("no service backend configured")
	}

	active, err := m.backend.IsActive(name)
	if err != nil {
		return nil, err
	}

	enabled, err := m.backend.IsEnabled(name)
	if err != nil {
		return nil, err
	}

	return &Service{
		Name:    name,
		Active:  active,
		Enabled: enabled,
	}, nil
}
