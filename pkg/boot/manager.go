package boot

// Manager handles boot management operations
type Manager struct {
	// bootloader interface?
}

// New creates a new boot manager
func New() *Manager {
	return &Manager{}
}

// Kernel represents a kernel
type Kernel struct {
	Name    string
	Version string
	Current bool
}

// List lists available kernels
func (m *Manager) List() ([]Kernel, error) {
	return nil, nil
}

// SetDefault sets the default kernel
func (m *Manager) SetDefault(kernel string) error {
	return nil
}
