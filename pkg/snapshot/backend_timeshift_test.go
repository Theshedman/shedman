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
			// Return valid JSON
			jsonOut := `{
				"name": "2023-01-01_12-00-00",
				"comments": "timeshift test",
				"created": 1672574400,
				"tags": "ondemand",
				"type": "rsync"
			}`
			return []byte(jsonOut), nil
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
	// ID maps to "name" in JSON
	if snap.ID != "2023-01-01_12-00-00" {
		t.Errorf("Expected ID '2023-01-01_12-00-00', got '%s'", snap.ID)
	}
	if snap.Backend != "timeshift" {
		t.Errorf("Expected backend 'timeshift', got '%s'", snap.Backend)
	}
}
