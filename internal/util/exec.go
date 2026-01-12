package util

import (
	"os/exec"
)

// Executor defines an interface for executing external commands.
// It abstracts os/exec to facilitate testing.
type Executor interface {
	// Command creates an exec.Cmd object.
	// Note: We cannot easily mock exec.Cmd methods directly as it's a struct.
	// So we prefer high-level methods like Run or Output for simple cases.
	// But PacmanSourceProvider uses Output().

	// Output runs the command and returns stdout.
	Output(name string, args ...string) ([]byte, error)
}

// RealExecutor implements Executor using os/exec.
type RealExecutor struct{}

func (e *RealExecutor) Output(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.Output()
}
