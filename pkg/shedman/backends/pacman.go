package backends

import (
"fmt"
"os/exec"
)

const DefaultPacmanBinary = "/usr/bin/pacman"

// CommandExecutor interface allows mocking command execution in tests
type CommandExecutor interface {
	Run(name string, args ...string) error
}

// RealCommandExecutor executes real system commands
type RealCommandExecutor struct{}

func (r *RealCommandExecutor) Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

type PacmanBackend struct {
	binaryPath string
	executor   CommandExecutor
}

// NewPacmanBackend creates a new PacmanBackend with default settings
func NewPacmanBackend() *PacmanBackend {
	return NewPacmanBackendWithExecutor(DefaultPacmanBinary, &RealCommandExecutor{})
}

// NewPacmanBackendWithExecutor creates a PacmanBackend with custom executor (for testing)
func NewPacmanBackendWithExecutor(binaryPath string, executor CommandExecutor) *PacmanBackend {
	return &PacmanBackend{
		binaryPath: binaryPath,
		executor:   executor,
	}
}

func (p *PacmanBackend) Name() string {
	return "pacman"
}

func (p *PacmanBackend) Sync() error {
	if err := p.executor.Run(p.binaryPath, "-Sy"); err != nil {
		return fmt.Errorf("failed to sync pacman databases: %w", err)
	}
	return nil
}
