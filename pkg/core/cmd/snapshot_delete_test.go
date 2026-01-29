package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
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

	// Execute
	if err := RunSnapshotDelete(context.Background(), engine, args, buf); err != nil {
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
