package util

import (
	"os/exec"
)

// Executor defines an interface for executing external commands.
// It abstracts os/exec to facilitate testing.
type Executor interface {
	// Command creates an exec.Cmd object.

	// Output runs the command and returns stdout.
	Output(name string, args ...string) ([]byte, error)
}

// RealExecutor implements Executor using os/exec.
type RealExecutor struct{}

func (e *RealExecutor) Output(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.Output()
}
