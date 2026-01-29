package restic

import (
	"context"
	"io"
	"os/exec"
	"testing"

	"github.com/theshedman/shedman/pkg/executor"
)

func TestResticManager_Init(t *testing.T) {
	var calledName string
	var calledArgs []string

	mockExec := &executor.MockExecutor{
		CommandFunc: func(name string, args ...string) *exec.Cmd {
			calledName = name
			calledArgs = args
			return exec.Command("echo", "created repo")
		},
	}
	mgr := NewManager(mockExec, "password")

	err := mgr.Init(context.Background(), "gdrive:repo", io.Discard)
	if err != nil {
		t.Errorf("Init failed: %v", err)
	}

	if calledName != "restic" {
		t.Errorf("Expected restic, got %s", calledName)
	}
	// Verify args: -r rclone:gdrive:repo init
	expected := []string{"-r", "rclone:gdrive:repo", "init"}
	if len(calledArgs) != len(expected) {
		t.Errorf("Args mismatch")
	}
}

func TestResticManager_List(t *testing.T) {
	mockExec := &executor.MockExecutor{
		CommandFunc: func(name string, args ...string) *exec.Cmd {
			// Validate JSON flag usage
			// Return mock JSON
			return exec.Command("echo", `[{"id":"a1b2c3d4","time":"2023-01-01T12:00:00Z","paths":["/data"],"tags":["snap-1"]}]`)
		},
	}
	mgr := NewManager(mockExec, "password")

	snaps, err := mgr.List(context.Background(), "gdrive:repo")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(snaps) != 1 {
		t.Errorf("Expected 1 snapshot, got %d", len(snaps))
	}
	if snaps[0].ID != "a1b2c3d4" {
		t.Errorf("Incorrect ID")
	}
}
