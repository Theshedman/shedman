package snapshot

import (
	"fmt"
	"strings"
	"time"

	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/internal/util"
)

// TimeshiftBackend implements SnapshotManager using the 'timeshift' CLI
type TimeshiftBackend struct {
	cfg  *config.Config
	exec util.Executor
}

// NewTimeshiftBackend creates a new timeshift backend
func NewTimeshiftBackend(cfg *config.Config, executor util.Executor) *TimeshiftBackend {
	return &TimeshiftBackend{
		cfg:  cfg,
		exec: executor,
	}
}

func (b *TimeshiftBackend) GetBackendName() string {
	return "timeshift"
}

// Create creates a new snapshot using 'timeshift --create'
func (b *TimeshiftBackend) Create(desc string, opts CreateOptions) (*Snapshot, error) {
	// timeshift --create --comments "desc" --tags "tags" --json
	args := []string{"--create", "--comments", desc}

	if opts.IncludeHome {
		// Log warning or handle config override if possible
	}

	if len(opts.Tags) > 0 {
		args = append(args, "--tags", strings.Join(opts.Tags, ","))
	} else if opts.Type != "" {
		args = append(args, "--tags", opts.Type)
	}

	_, err := b.exec.Output("timeshift", args...)
	if err != nil {
		return nil, fmt.Errorf("timeshift create failed: %w", err)
	}

	// For MVP TDD, assume success and return struct.
	// Real implementation needs output parsing or a follow-up list command.
	return &Snapshot{
		Description: desc,
		Timestamp:   time.Now(),
		Backend:     "timeshift",
		// ID would be parsed from 'out' in a real scenario
	}, nil
}

// Stubs for other interface methods
func (b *TimeshiftBackend) List(opts ListOptions) ([]Snapshot, error)    { return nil, nil }
func (b *TimeshiftBackend) Delete(id string) error                       { return nil }
func (b *TimeshiftBackend) Restore(id string, opts RestoreOptions) error { return nil }
func (b *TimeshiftBackend) Prune(opts PruneOptions) error                { return nil }
func (b *TimeshiftBackend) Push(id string, target RemoteTarget) error    { return nil }
func (b *TimeshiftBackend) Pull(id string, source RemoteTarget) error    { return nil }
func (b *TimeshiftBackend) Diff(id1, id2 string) (DiffResult, error)     { return DiffResult{}, nil }
