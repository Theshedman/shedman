package alpm

import (
	"os/exec"
)

// CommandExecutor interface (copied from core to avoid import)
type CommandExecutor interface {
	Run(name string, args ...string) error
	Output(name string, args ...string) ([]byte, error)
}

// RealExecutor implements CommandExecutor
type RealExecutor struct{}

func (e *RealExecutor) Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

func (e *RealExecutor) Output(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.Output()
}
