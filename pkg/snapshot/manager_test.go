package snapshot

import (
	"errors"
	"testing"
	"time"
)

// MockBackend for testing
type MockBackend struct {
	snapshots []*Snapshot
	createErr error
}

func (m *MockBackend) Name() string      { return "mock" }
func (m *MockBackend) IsAvailable() bool { return true }
func (m *MockBackend) Create(opts CreateOptions) (*Snapshot, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	snap := &Snapshot{
		ID:          "snap-1",
		Description: opts.Description,
		Timestamp:   time.Now(),
		Type:        opts.Type,
	}
	m.snapshots = append(m.snapshots, snap)
	return snap, nil
}
func (m *MockBackend) Restore(id string, opts RestoreOptions) error { return nil }
func (m *MockBackend) List() ([]*Snapshot, error)                   { return m.snapshots, nil }
func (m *MockBackend) Delete(id string) error                       { return nil }
func (m *MockBackend) Get(id string) (*Snapshot, error) {
	for _, s := range m.snapshots {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, errors.New("not found")
}

func TestManager_Create(t *testing.T) {
	mock := &MockBackend{}
	mgr := NewWithBackend(mock)

	opts := CreateOptions{
		Description: "test snapshot",
		Type:        "manual",
	}

	snap, err := mgr.Create(opts)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if snap.Description != opts.Description {
		t.Errorf("Expected description '%s', got '%s'", opts.Description, snap.Description)
	}

	if len(mock.snapshots) != 1 {
		t.Errorf("Expected 1 snapshot in backend, got %d", len(mock.snapshots))
	}
}

func TestManager_List(t *testing.T) {
	mock := &MockBackend{
		snapshots: []*Snapshot{
			{ID: "1", Description: "snap 1"},
			{ID: "2", Description: "snap 2"},
		},
	}
	mgr := NewWithBackend(mock)

	snaps, err := mgr.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(snaps) != 2 {
		t.Errorf("Expected 2 snapshots, got %d", len(snaps))
	}
}
