package executor

import (
	"context"
	"os/exec"
)

// MockExecutor implements Executor for testing
type MockExecutor struct {
	// Function hooks to allow per-test mocking
	OutputFunc  func(name string, args ...string) ([]byte, error)
	CommandFunc func(name string, args ...string) *exec.Cmd
}

func (m *MockExecutor) Output(name string, args ...string) ([]byte, error) {
	if m.OutputFunc != nil {
		return m.OutputFunc(name, args...)
	}
	return nil, nil
}

func (m *MockExecutor) Command(name string, args ...string) *exec.Cmd {
	if m.CommandFunc != nil {
		return m.CommandFunc(name, args...)
	}
	// Default to a command that does nothing but succeed
	return exec.Command("true")
}

func (m *MockExecutor) CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	// For now, delegate to Command (ignoring context for simple mocks)
	// or create a context-aware hook if needed.
	return m.Command(name, args...)
}
