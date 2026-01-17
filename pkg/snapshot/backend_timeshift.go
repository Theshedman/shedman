package snapshot

import (
	"encoding/json"
	"fmt"
	"os"
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

// TimeshiftJSON represents the structure of timeshift --json output
type TimeshiftJSON struct {
	Name     string `json:"name"`    // Date/Time ID
	Type     string `json:"type"`    // rsync / btrfs
	Created  int64  `json:"created"` // Timestamp
	Tags     string `json:"tags"`
	Comments string `json:"comments"`
	Device   string `json:"device"`
	UUID     string `json:"uuid"`
}

func (b *TimeshiftBackend) Create(desc string, opts CreateOptions) (*Snapshot, error) {
	args := []string{"--create", "--comments", desc, "--json"}

	if len(opts.Tags) > 0 {
		args = append(args, "--tags", strings.Join(opts.Tags, ","))
	} else if opts.Type != "" {
		args = append(args, "--tags", opts.Type)
	}

	out, err := b.exec.Output("timeshift", args...)
	if err != nil {
		return nil, fmt.Errorf("timeshift create failed: %w", err)
	}

	jsonStr := util.ExtractJSON(string(out))
	var tsSnap TimeshiftJSON
	if err := json.Unmarshal([]byte(jsonStr), &tsSnap); err != nil {
		return nil, fmt.Errorf("failed to parse timeshift output: %w", err)
	}

	return &Snapshot{
		ID:          tsSnap.Name,
		Description: tsSnap.Comments,
		Timestamp:   time.Unix(tsSnap.Created, 0),
		Backend:     "timeshift",
		Type:        tsSnap.Tags,
	}, nil
}

func (b *TimeshiftBackend) List(opts ListOptions) ([]Snapshot, error) {
	args := []string{"--list", "--json"}
	out, err := b.exec.Output("timeshift", args...)
	if err != nil {
		return nil, fmt.Errorf("timeshift list failed: %w", err)
	}

	jsonStr := util.ExtractJSON(string(out)) // Timeshift outputs array
	var tsSnaps []TimeshiftJSON
	if err := json.Unmarshal([]byte(jsonStr), &tsSnaps); err != nil {
		return nil, fmt.Errorf("failed to parse timeshift list: %w", err)
	}

	var snapshots []Snapshot
	for _, s := range tsSnaps {
		snapshots = append(snapshots, Snapshot{
			ID:          s.Name,
			Description: s.Comments,
			Type:        s.Tags,
			Timestamp:   time.Unix(s.Created, 0),
			Backend:     "timeshift",
			// Size is hard to get from basic list without --snapshot-device info
		})
	}
	return snapshots, nil
}

func (b *TimeshiftBackend) Delete(id string) error {
	_, err := b.exec.Output("timeshift", "--delete", "--snapshot", id)
	return err
}

func (b *TimeshiftBackend) Restore(id string, opts RestoreOptions) error {
	// Wrapper for timeshift restore
	return fmt.Errorf("interactive restore not supported via shedman yet; run 'timeshift --restore --snapshot %s' manually", id)
}

func (b *TimeshiftBackend) Prune(opts PruneOptions) error {

	snaps, err := b.List(ListOptions{})
	if err != nil {
		return err
	}

	if opts.KeepLast > 0 && len(snaps) > opts.KeepLast {
		toDelete := snaps[:len(snaps)-opts.KeepLast]
		for _, s := range toDelete {
			if err := b.Delete(s.ID); err != nil {
				return err
			}
		}
	}
	return nil
}
func (b *TimeshiftBackend) Push(id string, target RemoteTarget, opts RemoteOptions) error {
	localPath := fmt.Sprintf("/timeshift/snapshots/%s/", id)
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		localPath = fmt.Sprintf("/run/timeshift/backup/timeshift/snapshots/%s/", id)
		if _, err := os.Stat(localPath); os.IsNotExist(err) {
			return fmt.Errorf("snapshot %s path not found (checked /timeshift/snapshots and /run/timeshift/...)", id)
		}
	}

	remotePath := target.Path
	if !strings.HasSuffix(remotePath, "/") {
		remotePath += "/"
	}
	remotePath += id

	args := []string{"sync", "-P", localPath, remotePath}

	if opts.Delete {
		args = append(args, "--delete")
	}
	if opts.Bandwidth > 0 {
		args = append(args, "--bwlimit", fmt.Sprintf("%dk", opts.Bandwidth))
	}

	cmdArgs := util.GetPrivilegedRcloneCommand(args)

	fmt.Printf("Executing: %s\n", strings.Join(cmdArgs, " "))
	cmd := (&util.RealExecutor{}).Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rclone sync failed: %w", err)
	}

	return nil
}

func (b *TimeshiftBackend) Pull(id string, source RemoteTarget, opts RemoteOptions) error {
	localPath := fmt.Sprintf("/timeshift/snapshots/%s/", id)
	if _, err := os.Stat("/timeshift/snapshots"); os.IsNotExist(err) {
		if _, err := os.Stat("/run/timeshift/backup/timeshift/snapshots"); err == nil {
			localPath = fmt.Sprintf("/run/timeshift/backup/timeshift/snapshots/%s/", id)
		}
	}

	remotePath := source.Path
	if !strings.HasSuffix(remotePath, "/") {
		remotePath += "/"
	}
	remotePath += id

	args := []string{"sync", "-P", remotePath, localPath}

	if opts.Delete {
		args = append(args, "--delete")
	}
	if opts.Bandwidth > 0 {
		args = append(args, "--bwlimit", fmt.Sprintf("%dk", opts.Bandwidth))
	}

	cmdArgs := util.GetPrivilegedRcloneCommand(args)

	fmt.Printf("Executing: %s\n", strings.Join(cmdArgs, " "))
	cmd := (&util.RealExecutor{}).Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rclone sync failed: %w", err)
	}

	return nil
}
func (b *TimeshiftBackend) Diff(id1, id2 string) (DiffResult, error) { return DiffResult{}, nil }
