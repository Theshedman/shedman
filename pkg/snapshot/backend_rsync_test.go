package snapshot

import (
	"testing"

	"github.com/theshedman/shedman/internal/config"
)

func TestRsyncBackend_Create(t *testing.T) {
	cfg := config.Default()

	mockExec := &MockExecutor{
		OutputFunc: func(name string, args ...string) ([]byte, error) {
			if name != "rsync" {
				t.Errorf("Expected command 'rsync', got '%s'", name)
			}
			return []byte(""), nil // Helper commands return empty or success
		},
	}

	backend := NewRsyncBackend(cfg, mockExec)
	backend.SetRoot(t.TempDir()) // Use temp dir for directory creation

	snap, err := backend.Create("rsync test", CreateOptions{})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if snap.Description != "rsync test" {
		t.Errorf("Expected description 'rsync test', got '%s'", snap.Description)
	}
	if snap.Backend != "rsync" {
		t.Errorf("Expected backend 'rsync', got '%s'", snap.Backend)
	}
}
