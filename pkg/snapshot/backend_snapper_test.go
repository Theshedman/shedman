package snapshot

import (
	"testing"

	"github.com/theshedman/shedman/internal/config"
)

// MockExecutor implements util.Executor for testing
type MockExecutor struct {
	OutputFunc func(name string, args ...string) ([]byte, error)
}

func (m *MockExecutor) Output(name string, args ...string) ([]byte, error) {
	if m.OutputFunc != nil {
		return m.OutputFunc(name, args...)
	}
	return nil, nil
}

func TestSnapperBackend_Create(t *testing.T) {
	cfg := config.Default()

	mockExec := &MockExecutor{
		OutputFunc: func(name string, args ...string) ([]byte, error) {
			if name != "snapper" {
				t.Errorf("Expected command 'snapper', got '%s'", name)
			}
			if args[0] != "create" {
				t.Errorf("Expected subcommand 'create', got '%s'", args[0])
			}
			return []byte("42\n"), nil
		},
	}

	backend := NewSnapperBackend(cfg, mockExec)

	snap, err := backend.Create("test snapshot", CreateOptions{Type: "single"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if snap.ID != "42" {
		t.Errorf("Expected ID '42', got '%s'", snap.ID)
	}
	if snap.Backend != "snapper" {
		t.Errorf("Expected backend 'snapper', got '%s'", snap.Backend)
	}
}
