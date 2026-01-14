package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/snapshot"
)

func TestSnapshotPruneCmd(t *testing.T) {
	// Setup
	engine := core.NewEngine()
	mock := &MockSnapshotManager{}
	engine.SetSnapshotManager(mock)

	buf := new(bytes.Buffer)
	opts := snapshot.PruneOptions{KeepLast: 5}

	// Test Prune
	if err := RunSnapshotPrune(engine, opts, buf); err != nil {
		t.Fatalf("Prune execution failed: %v", err)
	}
	if !strings.Contains(buf.String(), "Prune completed") {
		t.Errorf("Expected success message, got: %s", buf.String())
	}
}
