package util

import (
	"fmt"
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
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("%w: %s", err, string(exitErr.Stderr))
		}
		return nil, err
	}
	return out, nil
}
