package snapshot

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/internal/util"
)

const (
	defaultRsyncStorage = "/var/lib/shedman/snapshots"
	rsyncDateFormat     = "20060102-150405"
)

// RsyncBackend implements SnapshotManager using custom rsync logic
type RsyncBackend struct {
	cfg  *config.Config
	exec util.Executor
	root string
}

// SetRoot sets the snapshot root directory (for testing)
func (b *RsyncBackend) SetRoot(path string) {
	b.root = path
}

// NewRsyncBackend creates a new rsync backend
func NewRsyncBackend(cfg *config.Config, executor util.Executor) *RsyncBackend {
	return &RsyncBackend{
		cfg:  cfg,
		exec: executor,
		root: defaultRsyncStorage,
	}
}

func (b *RsyncBackend) GetBackendName() string {
	return "rsync"
}

// Create creates a new snapshot using 'rsync'
func (b *RsyncBackend) Create(desc string, opts CreateOptions) (*Snapshot, error) {
	if err := os.MkdirAll(b.root, 0755); err != nil {
		return nil, fmt.Errorf("failed to create snapshot root: %w", err)
	}

	ts := time.Now()
	id := ts.Format(rsyncDateFormat)
	snapPath := filepath.Join(b.root, id)
	latestLink := filepath.Join(b.root, "latest")

	args := []string{"-a", "--delete"}

	if _, err := os.Lstat(latestLink); err == nil {
		args = append(args, "--link-dest="+latestLink)
	}

	excludes := []string{"/dev", "/proc", "/sys", "/tmp", "/run", "/mnt", "/media", "/lost+found", "/var/lib/shedman/snapshots"}
	for _, excl := range excludes {
		args = append(args, "--exclude="+excl)
	}

	args = append(args, "/", snapPath)

	_, err := b.exec.Output("rsync", args...)
	if err != nil {
		return nil, fmt.Errorf("rsync failed: %w", err)
	}

	os.Remove(latestLink)
	if err := os.Symlink(id, latestLink); err != nil {
	}

	return &Snapshot{
		ID:          id,
		Description: desc,
		Timestamp:   ts,
		Backend:     "rsync",
	}, nil
}

func (b *RsyncBackend) List(opts ListOptions) ([]Snapshot, error) {
	entries, err := os.ReadDir(b.root)
	if err != nil {
		if os.IsNotExist(err) {
			return []Snapshot{}, nil
		}
		return nil, fmt.Errorf("failed to list snapshot dir: %w", err)
	}

	var snapshots []Snapshot
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if name == "latest" {
			continue
		}

		// Parse timestamp
		ts, err := time.Parse(rsyncDateFormat, name)
		if err != nil {
			// Skip unknown folders
			continue
		}

		snapshots = append(snapshots, Snapshot{
			ID:          name,
			Description: "Rsync Snapshot",
			Timestamp:   ts,
			Backend:     "rsync",
			Type:        "single",
		})
	}

	// Sort reverse chronological
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Timestamp.After(snapshots[j].Timestamp)
	})

	return snapshots, nil
}

func (b *RsyncBackend) Delete(id string) error {
	path := filepath.Join(b.root, id)
	// Safety check: ensure path is within root
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(absPath, b.root) {
		return fmt.Errorf("invalid snapshot path")
	}

	return os.RemoveAll(path)
}

func (b *RsyncBackend) Restore(id string, opts RestoreOptions) error {
	snapPath := filepath.Join(b.root, id)
	if _, err := os.Stat(snapPath); os.IsNotExist(err) {
		return fmt.Errorf("snapshot %s not found", id)
	}

	// rsync -a /path/to/snap/ /
	args := []string{"-a", snapPath + "/", "/"}

	_, err := b.exec.Output("rsync", args...)
	return err
}

func (b *RsyncBackend) Prune(opts PruneOptions) error {
	snaps, err := b.List(ListOptions{})
	if err != nil {
		return err
	}

	if opts.KeepLast > 0 && len(snaps) > opts.KeepLast {
		toDelete := snaps[opts.KeepLast:] // snaps are sorted newest first
		for _, s := range toDelete {
			if err := b.Delete(s.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *RsyncBackend) Push(id string, target RemoteTarget, opts RemoteOptions) error {
	snapPath := filepath.Join(b.root, id)
	if _, err := os.Stat(snapPath); os.IsNotExist(err) {
		return fmt.Errorf("snapshot %s not found", id)
	}

	// rsync -avz --delete /path/to/snap/ user@host:/path/to/dest/
	args := []string{"-a", "-v", "-z"}
	if opts.Delete {
		args = append(args, "--delete")
	}
	if opts.Bandwidth > 0 {
		args = append(args, fmt.Sprintf("--bwlimit=%d", opts.Bandwidth))
	}

	args = append(args, snapPath+"/")

	dest := target.Path
	if target.Name != "" && target.Name != "local" {
		if strings.Contains(target.Name, ":") {
			dest = target.Name + ":" + target.Path
		} else {
			if target.Name != "local" {
				dest = target.Name + ":" + target.Path
			}
		}
	}
	args = append(args, dest)

	_, err := b.exec.Output("rsync", args...)
	return err
}

func (b *RsyncBackend) Pull(id string, source RemoteTarget, opts RemoteOptions) error {
	src := source.Path
	if source.Name != "" && source.Name != "local" {
		if strings.Contains(source.Name, ":") {
			src = source.Name + ":" + source.Path
		} else {
			src = source.Name + ":" + source.Path
		}
	}

	if !strings.HasSuffix(src, "/") {
		src += "/"
	}

	snapPath := filepath.Join(b.root, id)
	if err := os.MkdirAll(snapPath, 0755); err != nil {
		return fmt.Errorf("failed to create snapshot dir: %w", err)
	}

	args := []string{"-a", "-v", "-z"}
	if opts.Bandwidth > 0 {
		args = append(args, fmt.Sprintf("--bwlimit=%d", opts.Bandwidth))
	}

	args = append(args, src, snapPath+"/")

	_, err := b.exec.Output("rsync", args...)
	return err
}

func (b *RsyncBackend) Diff(id1, id2 string) (DiffResult, error) {
	path1 := filepath.Join(b.root, id1)
	path2 := filepath.Join(b.root, id2)

	if _, err := os.Stat(path1); os.IsNotExist(err) {
		return DiffResult{}, fmt.Errorf("snapshot %s not found", id1)
	}
	if _, err := os.Stat(path2); os.IsNotExist(err) {
		return DiffResult{}, fmt.Errorf("snapshot %s not found", id2)
	}

	args := []string{"-n", "-a", "-i", "--delete", path2 + "/", path1 + "/"}
	output, err := b.exec.Output("rsync", args...)
	if err != nil {
		return DiffResult{}, fmt.Errorf("diff failed: %w", err)
	}

	result := DiffResult{}
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if len(line) < 12 {
			continue
		}
		// Itemize format: YXcstpoguax  path/to/file
		// Y is update type: > (sent), < (recv), c (local change/creation)

		code := line[0]
		// skip checksum/size etc for rough parsing
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		path := parts[len(parts)-1]

		if strings.HasPrefix(line, "*deleting") {
			result.Removed = append(result.Removed, path)
			continue
		}

		if code == '>' {
			if strings.Contains(parts[0], "+++++++++") {
				result.Added = append(result.Added, path)
			} else {
				result.Modified = append(result.Modified, path)
			}
		}
	}

	return result, nil
}
