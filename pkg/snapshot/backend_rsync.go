package snapshot

import (
	"fmt"
	"time"

	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/internal/util"
)

// RsyncBackend implements SnapshotManager using custom rsync logic
type RsyncBackend struct {
	cfg  *config.Config
	exec util.Executor
}

// NewRsyncBackend creates a new rsync backend
func NewRsyncBackend(cfg *config.Config, executor util.Executor) *RsyncBackend {
	return &RsyncBackend{
		cfg:  cfg,
		exec: executor,
	}
}

func (b *RsyncBackend) GetBackendName() string {
	return "rsync"
}

// Create creates a new snapshot using 'rsync'
func (b *RsyncBackend) Create(desc string, opts CreateOptions) (*Snapshot, error) {
	// rsync -a --link-dest=... source dest
	// This is a complex logic, for TDD step 1 we just verify we can call the binary.

	// Placeholder: just run rsync --version to verify ability to execute
	_, err := b.exec.Output("rsync", "--version")
	if err != nil {
		return nil, fmt.Errorf("rsync execution failed: %w", err)
	}

	return &Snapshot{
		Description: desc,
		Timestamp:   time.Now(),
		Backend:     "rsync",
		ID:          fmt.Sprintf("rsync-%d", time.Now().Unix()),
	}, nil
}
func (b *RsyncBackend) List(opts ListOptions) ([]Snapshot, error)    { return nil, nil }
func (b *RsyncBackend) Delete(id string) error                       { return nil }
func (b *RsyncBackend) Restore(id string, opts RestoreOptions) error { return nil }
func (b *RsyncBackend) Prune(opts PruneOptions) error                { return nil }
func (b *RsyncBackend) Push(id string, target RemoteTarget) error    { return nil }
func (b *RsyncBackend) Pull(id string, source RemoteTarget) error    { return nil }
func (b *RsyncBackend) Diff(id1, id2 string) (DiffResult, error)     { return DiffResult{}, nil }
