package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/snapshot"
)

func TestSnapshotDeleteCmd(t *testing.T) {
	// Setup
	engine := core.NewEngine()
	deletedID := ""
	mockMgr := &MockSnapshotManager{
		DeleteFunc: func(id string) error {
			deletedID = id
			return nil
		},
	}
	engine.SetSnapshotManager(mockMgr)

	buf := new(bytes.Buffer)
	args := []string{"snap-to-delete"}
	opts := SnapshotDeleteOptions{}

	// Execute
	if err := RunSnapshotDelete(context.Background(), engine, args, opts, buf); err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}

	// Verify
	output := buf.String()
	if !strings.Contains(output, "deleted successfully") {
		t.Errorf("Expected success message, got: %s", output)
	}
	if deletedID != "snap-to-delete" {
		t.Errorf("Expected deleted ID 'snap-to-delete', got: %s", deletedID)
	}
}

func TestSnapshotDeleteOlderThan(t *testing.T) {
	engine := core.NewEngine()
	pruneCalled := false
	mockMgr := &MockSnapshotManager{
		PruneFunc: func(opts snapshot.PruneOptions) error {
			pruneCalled = true
			if opts.OlderThan == 0 {
				t.Error("expected older-than duration")
			}
			return nil
		},
	}
	engine.SetSnapshotManager(mockMgr)

	buf := new(bytes.Buffer)
	opts := SnapshotDeleteOptions{
		OlderThan: 24 * time.Hour,
	}

	if err := RunSnapshotDelete(context.Background(), engine, []string{}, opts, buf); err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}
	if !pruneCalled {
		t.Error("expected prune to be called")
	}
}
