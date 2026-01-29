package snapshot

import (
	"context"
	"testing"

	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/pkg/executor"
)

func TestRsyncBackend_Create(t *testing.T) {
	cfg := config.Default()

	mockExec := &executor.MockExecutor{
		OutputFunc: func(name string, args ...string) ([]byte, error) {
			if name != "rsync" {
				t.Errorf("Expected command 'rsync', got '%s'", name)
			}
			return []byte(""), nil // Helper commands return empty or success
		},
	}

	backend := NewRsyncBackend(cfg, mockExec)
	backend.SetRoot(t.TempDir()) // Use temp dir for directory creation

	snap, err := backend.Create(context.Background(), "rsync test", CreateOptions{})
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

func TestRsyncBackend_DryRun(t *testing.T) {
	cfg := config.Default()

	mockExec := &executor.MockExecutor{
		OutputFunc: func(name string, args ...string) ([]byte, error) {
			t.Errorf("Unexpected command execution in dry-run: %s %v", name, args)
			return nil, nil
		},
	}

	backend := NewRsyncBackend(cfg, mockExec)
	backend.SetRoot(t.TempDir())

	snap, err := backend.Create(context.Background(), "dry run test", CreateOptions{DryRun: true})
	if err != nil {
		t.Fatalf("DryRun Create failed: %v", err)
	}

	if snap.ID != "dry-run" {
		t.Errorf("Expected dry-run ID, got '%s'", snap.ID)
	}
}
