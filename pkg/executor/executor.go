package executor

import (
	"context"
	"fmt"
	"os/exec"
)

// Executor defines an interface for executing external commands.
// It abstracts os/exec to facilitate testing.
type Executor interface {
	// CommandContext creates an exec.Cmd object with context.
	CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd

	// Command creates an exec.Cmd object.
	Command(name string, args ...string) *exec.Cmd

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

// Command creates an exec.Cmd object (helper for interactive use).
func (e *RealExecutor) Command(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

// CommandContext creates an exec.Cmd object with context.
func (e *RealExecutor) CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}
