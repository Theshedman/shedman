package snapshot

import (
	"context"
	"os/exec"
	"testing"

	"github.com/theshedman/shedman/internal/config"
)

// MockExecutor implements util.Executor for testing
type MockExecutor struct {
	// Func fields to allow mocking per test
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
	return exec.Command(name, args...)
}

func (m *MockExecutor) CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}

func TestSnapperBackend_Create(t *testing.T) {
	cfg := config.Default()

	mockExec := &MockExecutor{
		OutputFunc: func(name string, args ...string) ([]byte, error) {
			if name != "snapper" {
				t.Errorf("Expected command 'snapper', got '%s'", name)
			}
			if len(args) > 0 {
				if args[0] == "--csvout" && args[1] == "list-configs" {
					return []byte("config,subvolume\nroot,/\n"), nil
				}
				if args[0] == "create" {
					return []byte("42\n"), nil
				}
			}
			return nil, nil
		},
	}

	backend := NewSnapperBackend(cfg, mockExec)

	snap, err := backend.Create("test snapshot", CreateOptions{Type: "single"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if snap.ID == "" || len(snap.ID) != 15 {
		t.Errorf("Expected valid timestamp ID, got '%s'", snap.ID)
	}
	if snap.Backend != "snapper" {
		t.Errorf("Expected backend 'snapper', got '%s'", snap.Backend)
	}
}

func TestSnapperBackend_DryRun(t *testing.T) {
	cfg := config.Default()
	mockExec := &MockExecutor{
		OutputFunc: func(name string, args ...string) ([]byte, error) {
			// Allow read-only config detection
			if len(args) > 0 && args[1] == "list-configs" {
				return []byte("config,subvolume\nroot,/\n"), nil
			}
			t.Errorf("Unexpected command execution in dry-run: %s %v", name, args)
			return nil, nil
		},
	}
	backend := NewSnapperBackend(cfg, mockExec)

	snap, err := backend.Create("dry run test", CreateOptions{DryRun: true})
	if err != nil {
		t.Fatalf("DryRun Create failed: %v", err)
	}
	if snap.ID != "dry-run" {
		t.Errorf("Expected dry-run ID, got '%s'", snap.ID)
	}
}
