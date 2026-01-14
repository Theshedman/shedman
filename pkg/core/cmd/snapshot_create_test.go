package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/snapshot"
)

// MockSnapshotManager for testing CLI
type MockSnapshotManager struct {
	CreateFunc         func(desc string, opts snapshot.CreateOptions) (*snapshot.Snapshot, error)
	ListFunc           func(opts snapshot.ListOptions) ([]snapshot.Snapshot, error)
	RestoreFunc        func(id string, opts snapshot.RestoreOptions) error
	DeleteFunc         func(id string) error
	DiffFunc           func(id1, id2 string) (snapshot.DiffResult, error)
	GetBackendNameFunc func() string
}

func (m *MockSnapshotManager) Create(desc string, opts snapshot.CreateOptions) (*snapshot.Snapshot, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(desc, opts)
	}
	return &snapshot.Snapshot{ID: "test-snap-1", Backend: "mock"}, nil
}

func (m *MockSnapshotManager) GetBackendName() string {
	if m.GetBackendNameFunc != nil {
		return m.GetBackendNameFunc()
	}
	return "mock"
}

func (m *MockSnapshotManager) List(opts snapshot.ListOptions) ([]snapshot.Snapshot, error) {
	if m.ListFunc != nil {
		return m.ListFunc(opts)
	}
	return nil, nil
}
func (m *MockSnapshotManager) Delete(id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

func (m *MockSnapshotManager) Restore(id string, opts snapshot.RestoreOptions) error {
	if m.RestoreFunc != nil {
		return m.RestoreFunc(id, opts)
	}
	return nil
}

func (m *MockSnapshotManager) Prune(opts snapshot.PruneOptions) error { return nil }
func (m *MockSnapshotManager) Push(id string, target snapshot.RemoteTarget, opts snapshot.RemoteOptions) error {
	return nil
}
func (m *MockSnapshotManager) Pull(id string, source snapshot.RemoteTarget, opts snapshot.RemoteOptions) error {
	return nil
}
func (m *MockSnapshotManager) Diff(id1, id2 string) (snapshot.DiffResult, error) {
	if m.DiffFunc != nil {
		return m.DiffFunc(id1, id2)
	}
	return snapshot.DiffResult{}, nil
}

func TestSnapshotCreateCmd(t *testing.T) {
	// Setup
	engine := core.NewEngine()
	mockMgr := &MockSnapshotManager{}
	engine.SetSnapshotManager(mockMgr)

	buf := new(bytes.Buffer)
	opts := snapshot.CreateOptions{Type: "pre"}
	args := []string{"my-snap"}

	// Execute
	if err := RunSnapshotCreate(engine, args, opts, buf); err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}

	// Verify
	output := buf.String()
	if !contains(output, "Snapshot created successfully") {
		t.Errorf("Expected success message, got: %s", output)
	}
	if !contains(output, "ID: test-snap-1") {
		t.Errorf("Expected ID, got: %s", output)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
