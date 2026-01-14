package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/snapshot"
)

func TestSnapshotRemoteCmd(t *testing.T) {
	// Setup
	engine := core.NewEngine()
	mock := &MockSnapshotManager{
		// Should use updated mock (if I updated it in shared file, which I did via full overwrite previously)
		// But I need to ensure Push/Pull are stubbed to return nil or track calls.
		// The shared mock in snapshot_create_test.go has:
		// func (m *MockSnapshotManager) Push(...) error { return nil }
		// So it should work out of the box for success case.
	}
	engine.SetSnapshotManager(mock)

	buf := new(bytes.Buffer)
	target := snapshot.RemoteTarget{Name: "s3"}

	// Test Push
	if err := RunSnapshotRemotePush(engine, "snap-1", target, buf); err != nil {
		t.Fatalf("Push execution failed: %v", err)
	}
	if !strings.Contains(buf.String(), "Push successful") {
		t.Errorf("Expected success message, got: %s", buf.String())
	}
}
