package boot

import (
	"fmt"
	"os/exec"

	"github.com/theshedman/shedman/pkg/core"
)

// Executor defines command execution for boot management
type Executor interface {
	CombinedOutput(name string, args ...string) ([]byte, error)
	LookPath(file string) (string, error)
}

// RealExecutor implements Executor using os/exec
type RealExecutor struct{}

func (e *RealExecutor) CombinedOutput(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

func (e *RealExecutor) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

// Manager handles boot management operations
type Manager struct {
	core *core.Engine
	exec Executor
}

// NewWithExecutor creates a new boot manager with a custom executor
func NewWithExecutor(c *core.Engine, exec Executor) *Manager {
	return &Manager{
		core: c,
		exec: exec,
	}
}

// New creates a new boot manager
func New(c *core.Engine) *Manager {
	return &Manager{
		core: c,
		exec: &RealExecutor{},
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

	// Check for systemd-boot
	if _, err := m.exec.LookPath("bootctl"); err == nil {
		out, err := m.exec.CombinedOutput("bootctl", "set-default", kernel+".conf")
		if err != nil {
			return fmt.Errorf("bootctl failed (entry '%s.conf' might not exist): %w\nOutput: %s", kernel, err, string(out))
		}
		return nil
	}

	// GRUB check
	if _, err := m.exec.LookPath("grub-set-default"); err == nil {
		return fmt.Errorf("GRUB configuration not yet automated; please use 'grub-set-default' manually")
	}

	return fmt.Errorf("no supported bootloader management tool found (bootctl/grub-set-default)")
}
