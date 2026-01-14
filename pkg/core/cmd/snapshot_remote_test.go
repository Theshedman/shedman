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
	mock := &MockSnapshotManager{}
	engine.SetSnapshotManager(mock)

	buf := new(bytes.Buffer)
	target := snapshot.RemoteTarget{Name: "s3"}

	// Test Push
	opts := snapshot.RemoteOptions{}
	if err := RunSnapshotRemotePush(engine, "snap-1", target, opts, buf); err != nil {
		t.Fatalf("Push execution failed: %v", err)
	}
	if !strings.Contains(buf.String(), "Push successful") {
		t.Errorf("Expected success message, got: %s", buf.String())
	}
}
