package snapshot

import (
	"context"
	"os/exec"
	"testing"

	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/pkg/executor"
)

func TestTimeshiftBackend_Create(t *testing.T) {
	cfg := config.Default()

	mockExec := &executor.MockExecutor{
		CommandFunc: func(name string, args ...string) *exec.Cmd {
			if name != "timeshift" {
				t.Errorf("Expected command 'timeshift', got '%s'", name)
			}
			if args[0] != "--create" {
				t.Errorf("Expected flag '--create', got '%s'", args[0])
			}
			// Return JSON via echo
			jsonOut := `{
				"name": "2023-01-01_12-00-00",
				"comments": "timeshift test",
				"created": 1672574400,
				"tags": "ondemand",
				"type": "rsync"
			}`
			return exec.Command("echo", jsonOut)
		},
	}

	backend := NewTimeshiftBackend(cfg, mockExec)

	snap, err := backend.Create(context.Background(), "timeshift test", CreateOptions{Type: "ondemand"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if snap.Description != "timeshift test" {
		t.Errorf("Expected description 'timeshift test', got '%s'", snap.Description)
	}
	// ID matches "name" field
	if snap.ID != "2023-01-01_12-00-00" {
		t.Errorf("Expected ID '2023-01-01_12-00-00', got '%s'", snap.ID)
	}
	if snap.Backend != "timeshift" {
		t.Errorf("Expected backend 'timeshift', got '%s'", snap.Backend)
	}
}

func TestTimeshiftBackend_Restore(t *testing.T) {
	cfg := config.Default()
	var commandsRun []string

	mockExec := &executor.MockExecutor{
		// Mock Output for existence check via list
		CommandFunc: func(name string, args ...string) *exec.Cmd {
			if name == "timeshift" && len(args) > 0 && args[0] == "--list" {
				// Mock list output containing our target snapshot ID
				return exec.Command("echo", `[{"name":"snap-id","created":1234567890}]`)
			}
			cmdLine := name
			for _, arg := range args {
				cmdLine += " " + arg
			}
			commandsRun = append(commandsRun, cmdLine)
			return exec.Command("true")
		},
	}

	backend := NewTimeshiftBackend(cfg, mockExec)

	// Since we mock exec.Command to be "true", Run() will succeed
	err := backend.Restore(context.Background(), "snap-id", RestoreOptions{})
	if err != nil {
		t.Errorf("Restore failed: %v", err)
	}

	expectedArg := "--restore"
	found := false
	for _, c := range commandsRun {
		// Verify restore command call
		if c == "timeshift --restore --snapshot snap-id" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected interactive command with '%s', got %v", expectedArg, commandsRun)
	}
}

func TestTimeshiftBackend_DryRun(t *testing.T) {
	cfg := config.Default()
	mockExec := &executor.MockExecutor{
		OutputFunc: func(name string, args ...string) ([]byte, error) {
			t.Errorf("Unexpected command execution in dry-run: %s %v", name, args)
			return nil, nil
		},
	}
	backend := NewTimeshiftBackend(cfg, mockExec)

	snap, err := backend.Create(context.Background(), "dry run test", CreateOptions{DryRun: true})
	if err != nil {
		t.Fatalf("DryRun Create failed: %v", err)
	}
	if snap.ID != "dry-run" {
		t.Errorf("Expected dry-run ID, got '%s'", snap.ID)
	}
}
