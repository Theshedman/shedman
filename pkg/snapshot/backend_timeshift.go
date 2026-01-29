package snapshot

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/internal/util"
	"github.com/theshedman/shedman/pkg/executor"
)

// TimeshiftBackend implements SnapshotManager using the 'timeshift' CLI
type TimeshiftBackend struct {
	cfg *config.Config

	exec executor.Executor
}

// NewTimeshiftBackend creates a new timeshift backend
func NewTimeshiftBackend(cfg *config.Config, executor executor.Executor) *TimeshiftBackend {

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

func (b *TimeshiftBackend) Create(ctx context.Context, desc string, opts CreateOptions) (*Snapshot, error) {
	args := []string{"--create", "--comments", desc, "--json"}

	if len(opts.Tags) > 0 {
		args = append(args, "--tags", strings.Join(opts.Tags, ","))
	} else if opts.Type != "" {
		args = append(args, "--tags", opts.Type)
	}

	if opts.DryRun {
		fmt.Printf("Dry-run: %s %v\n", "timeshift", args)
		return &Snapshot{
			ID:          "dry-run",
			Description: desc,
			Timestamp:   time.Now(),
			Backend:     "timeshift",
			Type:        opts.Type,
		}, nil
	}

	out, err := b.exec.CommandContext(ctx, "timeshift", args...).Output()
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

func (b *TimeshiftBackend) List(ctx context.Context, opts ListOptions) ([]Snapshot, error) {
	args := []string{"--list", "--json"}
	out, err := b.exec.CommandContext(ctx, "timeshift", args...).Output()
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

func (b *TimeshiftBackend) Delete(ctx context.Context, id string) error {
	_, err := b.exec.CommandContext(ctx, "timeshift", "--delete", "--snapshot", id).Output()
	return err
}

func (b *TimeshiftBackend) Restore(ctx context.Context, id string, opts RestoreOptions) error {
	// Verify snapshot exists first
	snaps, err := b.List(ctx, ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list snapshots for verification: %w", err)
	}

	exists := false
	for _, s := range snaps {
		if s.ID == id {
			exists = true
			break
		}
	}
	if !exists {
		return fmt.Errorf("snapshot %s not found", id)
	}

	// Timeshift restore is interactive (ncurses), so we must attach stdin/stdout
	fmt.Printf("Launching Timeshift restore for snapshot %s...\n", id)
	fmt.Println("Warning: This requires root and may reboot your system.")

	if opts.DryRun {
		fmt.Printf("Dry-run: %s %s %s %s\n", "timeshift", "--restore", "--snapshot", id)
		return nil
	}

	cmd := b.exec.CommandContext(ctx, "timeshift", "--restore", "--snapshot", id)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("timeshift restore failed: %w", err)
	}

	return nil
}

func (b *TimeshiftBackend) Prune(ctx context.Context, opts PruneOptions) error {
	snaps, err := b.List(ctx, ListOptions{})
	if err != nil {
		return err
	}

	if opts.KeepLast > 0 && len(snaps) > opts.KeepLast {
		toDelete := snaps[:len(snaps)-opts.KeepLast]
		for _, s := range toDelete {
			if err := b.Delete(ctx, s.ID); err != nil {
				return err
			}
		}
	}
	return nil
}
func (b *TimeshiftBackend) Push(ctx context.Context, id string, target RemoteTarget, opts RemoteOptions) error {
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

	_, _ = fmt.Printf("Executing: %s\n", strings.Join(cmdArgs, " "))

	cmd := (&executor.RealExecutor{}).CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rclone sync failed: %w", err)
	}

	return nil
}

func (b *TimeshiftBackend) Pull(ctx context.Context, id string, source RemoteTarget, opts RemoteOptions) error {
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

	_, _ = fmt.Printf("Executing: %s\n", strings.Join(cmdArgs, " "))

	cmd := (&executor.RealExecutor{}).CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rclone sync failed: %w", err)
	}

	return nil
}
func (b *TimeshiftBackend) Diff(ctx context.Context, id1, id2 string) (DiffResult, error) {
	return DiffResult{}, nil
}
