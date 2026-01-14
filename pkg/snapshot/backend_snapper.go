package snapshot

import (
	"encoding/csv"
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

	r := csv.NewReader(strings.NewReader(string(out)))
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse snapper CSV: %w", err)
	}

	var snapshots []Snapshot
	for _, record := range records {
		if len(record) < 5 {
			continue
		}
		if record[0] == "number" || record[0] == "#" {
			continue
		}

		id := strings.TrimSpace(record[0])
		typ := strings.TrimSpace(record[1])
		dateStr := strings.TrimSpace(record[2])
		sizeStr := strings.TrimSpace(record[3])
		desc := strings.TrimSpace(record[4])

		if id == "0" && typ == "single" {
		}

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
func (b *SnapperBackend) Prune(opts PruneOptions) error {
	snaps, err := b.List(ListOptions{})
	if err != nil {
		return err
	}

	// This is a naive implementation:
	if opts.KeepLast > 0 && len(snaps) > opts.KeepLast {
		// Identify deletion candidates
		// Assume sorted chronological.
		toDelete := snaps[:len(snaps)-opts.KeepLast]
		for _, s := range toDelete {
			// Skip snapshot 0 (current)
			if s.ID == "0" {
				continue
			}
			if err := b.Delete(s.ID); err != nil {
				return fmt.Errorf("failed to prune snapshot %s: %w", s.ID, err)
			}
		}
	}

	return nil
}

func (b *SnapperBackend) Push(id string, target RemoteTarget, opts RemoteOptions) error {
	return fmt.Errorf("snapper backend does not support direct push yet")
}

func (b *SnapperBackend) Pull(id string, source RemoteTarget, opts RemoteOptions) error {
	return fmt.Errorf("snapper backend does not support direct pull yet")

}

func (b *SnapperBackend) Diff(id1, id2 string) (DiffResult, error) {
	// snapper status id1..id2
	args := []string{"status", fmt.Sprintf("%s..%s", id1, id2)}

	out, err := b.exec.Output("snapper", args...)
	if err != nil {
		return DiffResult{}, fmt.Errorf("snapper diff failed: %w", err)
	}

	// Parse output:
	// + File
	// - File
	// c File

	res := DiffResult{}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		status := parts[0]
		file := strings.Join(parts[1:], " ")

		if strings.HasPrefix(status, "+") {
			res.Added = append(res.Added, file)
		} else if strings.HasPrefix(status, "-") {
			res.Removed = append(res.Removed, file)
		} else if strings.HasPrefix(status, "c") {
			res.Modified = append(res.Modified, file)
		}
	}

	return res, nil
}
