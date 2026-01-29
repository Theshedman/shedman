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

func TestSnapshotListCmd(t *testing.T) {
	// Setup
	engine := core.NewEngine()
	// Using MockSnapshotManager defined in snapshot_create_test.go
	mockMgr := &MockSnapshotManager{
		ListFunc: func(opts snapshot.ListOptions) ([]snapshot.Snapshot, error) {
			return []snapshot.Snapshot{
				{
					ID:          "1",
					Description: "Snap 1",
					Timestamp:   time.Now(),
					Backend:     "mock",
					Size:        104857600, // 100MB
				},
				{
					ID:          "2",
					Description: "Snap 2",
					Timestamp:   time.Now(),
					Backend:     "mock",
					Size:        209715200, // 200MB
				},
			}, nil
		},
	}
	engine.SetSnapshotManager(mockMgr)

	buf := new(bytes.Buffer)
	opts := snapshot.ListOptions{}

	// Execute
	if err := RunSnapshotList(context.Background(), engine, opts, buf); err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}

	// Verify
	output := buf.String()
	// Should contain headers and mock data
	if !strings.Contains(output, "ID") || !strings.Contains(output, "DESCRIPTION") {
		t.Errorf("Expected table headers, got: %s", output)
	}
	if !strings.Contains(output, "Snap 1") || !strings.Contains(output, "Snap 2") {
		t.Errorf("Expected snapshot descriptions, got: %s", output)
	}
}
