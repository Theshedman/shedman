package snapshot

import (
	"testing"

	"github.com/theshedman/shedman/internal/config"
)

func TestTimeshiftBackend_Create(t *testing.T) {
	cfg := config.Default()

	mockExec := &MockExecutor{
		OutputFunc: func(name string, args ...string) ([]byte, error) {
			if name != "timeshift" {
				t.Errorf("Expected command 'timeshift', got '%s'", name)
			}
			if args[0] != "--create" {
				t.Errorf("Expected flag '--create', got '%s'", args[0])
			}
			return []byte("Snapshot created successfully"), nil
		},
	}

	backend := NewTimeshiftBackend(cfg, mockExec)

	snap, err := backend.Create("timeshift test", CreateOptions{Type: "ondemand"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if snap.Description != "timeshift test" {
		t.Errorf("Expected description 'timeshift test', got '%s'", snap.Description)
	}
	if snap.Backend != "timeshift" {
		t.Errorf("Expected backend 'timeshift', got '%s'", snap.Backend)
	}
}
