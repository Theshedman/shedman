package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/snapshot"
)

func TestSnapshotRestoreCmd(t *testing.T) {
	// Setup
	engine := core.NewEngine()
	// Using MockSnapshotManager defined in snapshot_create_test.go
	// We need to support RestoreFunc in it.
	// Parallel edit limitation in mocks requires sequential checks
	// I'll define a local mock or ensure the shared mock is updated.

	restoredID := ""
	mockMgr := &MockSnapshotManager{
		RestoreFunc: func(id string, opts snapshot.RestoreOptions) error {
			restoredID = id
			return nil
		},
	}
	engine.SetSnapshotManager(mockMgr)

	buf := new(bytes.Buffer)
	opts := snapshot.RestoreOptions{}
	args := []string{"snap-123"}

	// Execute
	if err := RunSnapshotRestore(context.Background(), engine, args, opts, buf); err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}

	// Verify
	output := buf.String()
	if !strings.Contains(output, "restored successfully") {
		t.Errorf("Expected success message, got: %s", output)
	}
	if restoredID != "snap-123" {
		t.Errorf("Expected restored ID 'snap-123', got: %s", restoredID)
	}
}
