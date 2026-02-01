package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/snapshot"
)

func TestUpdateCmd_Flags(t *testing.T) {
	cmd := NewUpdateCmd()

	tests := []struct {
		name      string
		flag      string
		shorthand string
	}{
		{"yes", "yes", "y"},
		{"shedos", "shedos", ""},
		{"official", "official", ""},
		{"aur", "aur", ""},
		{"refresh", "refresh", ""},
		{"delta", "delta", ""},
		{"limit-rate", "limit-rate", ""},
		{"retry", "retry", ""},
		{"timeout", "timeout", ""},
		{"ignore", "ignore", ""},
		{"ignoregroup", "ignoregroup", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := cmd.Flags().Lookup(tt.flag)
			if f == nil {
				t.Fatalf("Flag --%s not found", tt.flag)
			}
			if tt.shorthand != "" && f.Shorthand != tt.shorthand {
				t.Errorf("Flag --%s shorthand got %s, want %s", tt.flag, f.Shorthand, tt.shorthand)
			}
		})
	}
}

func TestUpdateCmd_Run(t *testing.T) {
	// This test just ensures the command is runnable/callable
	// Logic testing might require mocking backends which is harder in this CLI structure
	cmd := NewUpdateCmd()

	if cmd.Use != "update [packages...]" {
		t.Errorf("Expected Use 'update [packages...]', got '%s'", cmd.Use)
	}
}

func TestRunUpdate(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	// Mock behavior
	syncCalled := false
	mock.SyncFunc = func() error {
		syncCalled = true
		return nil
	}

	upgradeCalled := false
	mock.UpgradeFunc = func(pkgs []string, options core.UpgradeOptions) error {
		upgradeCalled = true
		if len(pkgs) != 1 || pkgs[0] != "test-pkg" {
			t.Errorf("Expected upgrade of [test-pkg], got %v", pkgs)
		}
		if !options.NoConfirm {
			t.Error("Expected NoConfirm=true")
		}
		return nil
	}

	var buf bytes.Buffer
	pkgs := []string{"test-pkg"}
	opts := core.UpgradeOptions{
		NoConfirm: true,
	}

	// TDD: RunUpdate doesn't exist yet
	if err := RunUpdate(eng, &buf, pkgs, opts); err != nil {
		t.Fatalf("RunUpdate failed: %v", err)
	}

	if !syncCalled {
		t.Error("Backend.Sync was not called")
	}
	if !upgradeCalled {
		t.Error("Backend.Upgrade was not called")
	}

	output := buf.String()
	if !strings.Contains(output, "Starting full system upgrade") {
		t.Errorf("Expected output to contain 'Starting full system upgrade', got: %s", output)
	}
}

func TestMergeUnique(t *testing.T) {
	out := mergeUnique([]string{"a", "b"}, []string{"b", "c"}, []string{"", "  d  "})
	expected := []string{"a", "b", "c", "d"}

	if len(out) != len(expected) {
		t.Fatalf("mergeUnique length = %d, want %d", len(out), len(expected))
	}
	for i := range expected {
		if out[i] != expected[i] {
			t.Errorf("mergeUnique[%d] = %s, want %s", i, out[i], expected[i])
		}
	}
}

type mockSnapshotManager struct {
	calls []string
}

func (m *mockSnapshotManager) Create(ctx context.Context, desc string, opts snapshot.CreateOptions) (*snapshot.Snapshot, error) {
	m.calls = append(m.calls, "snapshot:"+desc+":"+opts.Type)
	return &snapshot.Snapshot{ID: "snap-1"}, nil
}
func (m *mockSnapshotManager) List(ctx context.Context, opts snapshot.ListOptions) ([]snapshot.Snapshot, error) {
	return nil, nil
}
func (m *mockSnapshotManager) Delete(ctx context.Context, id string) error {
	return nil
}
func (m *mockSnapshotManager) Restore(ctx context.Context, id string, opts snapshot.RestoreOptions) error {
	return nil
}
func (m *mockSnapshotManager) Prune(ctx context.Context, opts snapshot.PruneOptions) error {
	return nil
}
func (m *mockSnapshotManager) Push(ctx context.Context, id string, target snapshot.RemoteTarget, opts snapshot.RemoteOptions) error {
	return nil
}
func (m *mockSnapshotManager) Pull(ctx context.Context, id string, source snapshot.RemoteTarget, opts snapshot.RemoteOptions) error {
	return nil
}
func (m *mockSnapshotManager) Diff(ctx context.Context, id1, id2 string) (snapshot.DiffResult, error) {
	return snapshot.DiffResult{}, nil
}
func (m *mockSnapshotManager) GetBackendName() string {
	return "mock"
}

func TestRunUpdate_AutoSnapshotBeforeUpdate(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	cfg := config.Default()
	cfg.Snapshot.AutoBeforeUpdate = true
	eng.SetConfig(cfg)

	snap := &mockSnapshotManager{}
	eng.SetSnapshotManager(snap)

	mock.SyncFunc = func() error {
		snap.calls = append(snap.calls, "sync")
		return nil
	}
	mock.UpgradeFunc = func(pkgs []string, options core.UpgradeOptions) error {
		snap.calls = append(snap.calls, "upgrade")
		return nil
	}

	var buf bytes.Buffer
	if err := RunUpdate(eng, &buf, nil, core.UpgradeOptions{}); err != nil {
		t.Fatalf("RunUpdate failed: %v", err)
	}

	if len(snap.calls) == 0 || snap.calls[0] != "snapshot:pre-update:pre" {
		t.Fatalf("expected snapshot first, got %v", snap.calls)
	}
}

func TestRunUpdate_AutoSnapshotSkippedWhenDisabled(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	cfg := config.Default()
	cfg.Snapshot.AutoBeforeUpdate = false
	eng.SetConfig(cfg)

	snap := &mockSnapshotManager{}
	eng.SetSnapshotManager(snap)

	mock.SyncFunc = func() error { return nil }
	mock.UpgradeFunc = func(pkgs []string, options core.UpgradeOptions) error { return nil }

	var buf bytes.Buffer
	if err := RunUpdate(eng, &buf, nil, core.UpgradeOptions{}); err != nil {
		t.Fatalf("RunUpdate failed: %v", err)
	}

	if len(snap.calls) != 0 {
		t.Fatalf("expected no snapshot calls, got %v", snap.calls)
	}
}
