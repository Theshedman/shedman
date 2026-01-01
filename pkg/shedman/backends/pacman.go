package backends

import (
	"fmt"
	"os"
	"os/exec"
)

const DefaultPacmanBinary = "/usr/bin/pacman"

// CommandExecutor interface allows mocking command execution in tests
type CommandExecutor interface {
	Run(name string, args ...string) error
}

// RealCommandExecutor executes real system commands
type RealCommandExecutor struct{}

type RootChecker interface {
    IsRoot() bool
}

func (r *RealCommandExecutor) Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

type RealRootChecker struct{}

func (r *RealRootChecker) IsRoot() bool {
    return os.Geteuid() == 0
}

type PacmanBackend struct {
	binaryPath string
	executor   CommandExecutor
    rootChecker RootChecker
}

// NewPacmanBackend creates a new PacmanBackend with default settings
func NewPacmanBackend() *PacmanBackend {
	return NewPacmanBackendWithExecutor(DefaultPacmanBinary, &RealCommandExecutor{}, &RealRootChecker{})
}

// NewPacmanBackendWithExecutor creates a PacmanBackend with custom executor (for testing)
func NewPacmanBackendWithExecutor(binaryPath string, executor CommandExecutor, rootChecker RootChecker) *PacmanBackend {
	return &PacmanBackend{
		binaryPath: binaryPath,
		executor:   executor,
	    rootChecker: rootChecker,
	}
}

func (p *PacmanBackend) Name() string {
	return "pacman"
}

func (p *PacmanBackend) Sync() error {
    if !p.rootChecker.IsRoot() {
        return fmt.Errorf("pacman sync requires root privileges")
    }

	if err := p.executor.Run(p.binaryPath, "-Sy"); err != nil {
		return fmt.Errorf("failed to sync pacman databases: %w", err)
	}
	
	return nil
}
