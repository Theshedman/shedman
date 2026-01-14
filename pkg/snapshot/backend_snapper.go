package snapshot

import (
	"fmt"
	"strings"
	"time"

	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/internal/util"
)

// SnapperBackend implements SnapshotManager using the 'snapper' CLI
type SnapperBackend struct {
	cfg  *config.Config
	exec util.Executor
}

// NewSnapperBackend creates a new snapper backend
func NewSnapperBackend(cfg *config.Config, executor util.Executor) *SnapperBackend {
	return &SnapperBackend{
		cfg:  cfg,
		exec: executor,
	}
}

func (b *SnapperBackend) GetBackendName() string {
	return "snapper"
}

// Create creates a new snapshot using 'snapper create'
func (b *SnapperBackend) Create(desc string, opts CreateOptions) (*Snapshot, error) {
	args := []string{"create", "--description", desc}

	if opts.Type != "" {
		args = append(args, "--type", opts.Type)
	} else {
		args = append(args, "--type", "single")
	}

	args = append(args, "--print-number")

	out, err := b.exec.Output("snapper", args...)
	if err != nil {
		if strings.Contains(err.Error(), "No permissions") {
			return nil, fmt.Errorf("snapper create failed (try running with sudo): %w", err)
		}
		return nil, fmt.Errorf("snapper create failed: %w", err)
	}

	id := strings.TrimSpace(string(out))

	return &Snapshot{
		ID:          id,
		Description: desc,
		Timestamp:   time.Now(),
		Type:        opts.Type,
		Backend:     "snapper",
	}, nil
}

// List lists snapshots
func (b *SnapperBackend) List(opts ListOptions) ([]Snapshot, error) {
	args := []string{"--csvout", "--iso", "list", "--columns", "number,type,date,used-space,description"}

	out, err := b.exec.Output("snapper", args...)
	if err != nil {
		if strings.Contains(err.Error(), "No permissions") {
			return nil, fmt.Errorf("snapper list failed (try running with sudo): %w", err)
		}
		return nil, fmt.Errorf("snapper list failed: %w", err)
	}

	var snapshots []Snapshot
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "number,") {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) < 5 {
			continue
		}

		id := strings.TrimSpace(parts[0])
		typ := strings.TrimSpace(parts[1])
		dateStr := strings.TrimSpace(parts[2])
		sizeStr := strings.TrimSpace(parts[3])
		desc := strings.Join(parts[4:], ",")
		desc = strings.Trim(desc, "\"") // Remove quotes if present

		var ts time.Time
		if t, err := time.Parse("2006-01-02 15:04:05", dateStr); err == nil {
			ts = t
		} else {
			ts = time.Now()
		}

		size := util.ParseSize(sizeStr)

		snapshots = append(snapshots, Snapshot{
			ID:          id,
			Description: desc,
			Type:        typ,
			Timestamp:   ts,
			Size:        size,
			Backend:     "snapper",
		})
	}

	return snapshots, nil
}

// Delete deletes a snapshot
func (b *SnapperBackend) Delete(id string) error {
	_, err := b.exec.Output("snapper", "delete", id)
	return err
}

// Restore restores a snapshot
func (b *SnapperBackend) Restore(id string, opts RestoreOptions) error {
	_, err := b.exec.Output("snapper", "rollback", id)
	return err
}
func (b *SnapperBackend) Prune(opts PruneOptions) error             { return nil }
func (b *SnapperBackend) Push(id string, target RemoteTarget) error { return nil }
func (b *SnapperBackend) Pull(id string, source RemoteTarget) error { return nil }
func (b *SnapperBackend) Diff(id1, id2 string) (DiffResult, error)  { return DiffResult{}, nil }
