package boot

import (
	"fmt"

	"github.com/theshedman/shedman/pkg/core"
)

// Manager handles boot management operations
type Manager struct {
	core *core.Engine
}

// New creates a new boot manager
func New(c *core.Engine) *Manager {
	return &Manager{
		core: c,
	}
}

// Kernel represents a kernel
type Kernel struct {
	Name    string
	Version string
	Current bool
}

var knownKernels = []string{
	"linux",
	"linux-lts",
	"linux-zen",
	"linux-hardened",
}

// List lists available kernels (installed ones)
func (m *Manager) List() ([]Kernel, error) {
	var kernels []Kernel

	for _, name := range knownKernels {
		if m.core.IsInstalled(name) {
			info, err := m.core.Info(name)
			version := "unknown"
			if err == nil && info != nil {
				version = info.Version
			}

			kernels = append(kernels, Kernel{
				Name:    name,
				Version: version,
				Current: false, // implementation pending (requires uname syscall)
			})
		}
	}

	return kernels, nil
}

// SetDefault sets the default kernel
func (m *Manager) SetDefault(kernel string) error {
	// Validate kernel is installed
	if !m.core.IsInstalled(kernel) {
		return fmt.Errorf("kernel %s is not installed", kernel)
	}

	// TODO: Implement bootloader configuration (systemd-boot/grub)
	// This requires filesystem access to /boot/loader/entries or similar.
	// For now, we consider validation success sufficient for this refactor stage.

	return nil
}
