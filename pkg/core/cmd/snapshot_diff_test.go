package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/snapshot"
)

func TestSnapshotDiffCmd(t *testing.T) {
	// Setup
	engine := core.NewEngine()
	mock := &MockSnapshotManager{}

	// Dynamically assign function if the field exists (it should based on previous edits)
	mock.DiffFunc = func(id1, id2 string) (snapshot.DiffResult, error) {
		return snapshot.DiffResult{
			Added:   []string{"pkg-new"},
			Removed: []string{"pkg-old"},
		}, nil
	}

	engine.SetSnapshotManager(mock)

	buf := new(bytes.Buffer)
	args := []string{"id1", "id2"}

	if err := RunSnapshotDiff(context.Background(), engine, args[0], args[1], buf); err != nil {
		t.Fatalf("Diff execution failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "pkg-new") {
		t.Errorf("Expected added package, got: %s", output)
	}
	if !strings.Contains(output, "pkg-old") {
		t.Errorf("Expected removed package, got: %s", output)
	}
}
