package snapshot

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/internal/output"
	"github.com/theshedman/shedman/internal/util"
	"github.com/theshedman/shedman/pkg/executor"
	"github.com/theshedman/shedman/pkg/snapshot/restic"
)

// SnapperBackend implements SnapshotManager using the 'snapper' CLI
type SnapperBackend struct {
	cfg  *config.Config
	exec executor.Executor
}

// NewSnapperBackend creates a new snapper backend
func NewSnapperBackend(cfg *config.Config, executor executor.Executor) *SnapperBackend {

	return &SnapperBackend{
		cfg:  cfg,
		exec: executor,
	}
}

// runWithSudo wraps the command with sudo if not running as root
func (b *SnapperBackend) runWithSudo(ctx context.Context, args ...string) ([]byte, error) {
	if os.Geteuid() != 0 {
		sudoArgs := append([]string{"-n", "snapper"}, args...) // -n: non-interactive
		// Use CommandContext
		cmd := b.exec.CommandContext(ctx, "sudo", sudoArgs...)
		// Do NOT bind Stdin, to prevent hanging on password prompt
		cmd.Stderr = os.Stderr
		return cmd.Output()
	}
	// Direct execution with context
	cmd := b.exec.CommandContext(ctx, "snapper", args...)
	return cmd.Output()
}

func (b *SnapperBackend) GetBackendName() string {
	return BackendSnapper
}

func (b *SnapperBackend) getConfigs(ctx context.Context) (map[string]string, error) {
	args := []string{"--csvout", "list-configs", "--columns", "config,subvolume"}
	out, err := b.runWithSudo(ctx, args...)
	if err != nil {
		return nil, err
	}
	r := csv.NewReader(strings.NewReader(string(out)))
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	configs := make(map[string]string)
	for _, rec := range records {
		if len(rec) >= 2 && rec[0] != "config" {
			name := strings.TrimSpace(rec[0])
			subvol := strings.TrimSpace(rec[1])
			configs[name] = subvol
		}
	}
	return configs, nil
}

// Create creates a new snapshot using 'snapper create'
func (b *SnapperBackend) Create(ctx context.Context, desc string, opts CreateOptions) (*Snapshot, error) {
	unifiedID := time.Now().Format("20060102-150405")

	targets := opts.TargetConfigs
	if len(targets) == 0 {
		configMap, err := b.getConfigs(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to detect snapper configs: %w", err)
		}
		for k := range configMap {
			targets = append(targets, k)
		}
	}

	successCount := 0
	var lastErr error

	output.Info("Creating unified snapshot %s for targets: %v", unifiedID, targets)

	for _, target := range targets {
		args := []string{"create", "--description", desc, "--userdata", fmt.Sprintf("shedman_id=%s", unifiedID), "-c", target}

		if opts.Type != "" {
			args = append(args, "--type", opts.Type)
		} else {
			args = append(args, "--type", "single")
		}

		// Cleanup logic if needed
		args = append(args, "--cleanup-algorithm", "number")

		if opts.DryRun {
			output.Info("Dry-run: %s %v", "snapper", args)
			successCount++
			continue
		}

		if _, err := b.runWithSudo(ctx, args...); err != nil {
			lastErr = err
			output.Warning("failed to snapshot target '%s': %v", target, err)
		} else {
			successCount++
		}
	}

	if successCount == 0 && len(targets) > 0 {
		return nil, fmt.Errorf("failed to create any snapshots. Last error: %w", lastErr)
	}

	if opts.DryRun {
		unifiedID = "dry-run"
	}

	return &Snapshot{
		ID:          unifiedID,
		Description: desc,
		Timestamp:   time.Now(),
		Type:        opts.Type,
		Backend:     BackendSnapper,
	}, nil
}

type snapperMatch struct {
	ID        string
	Config    string
	Subvolume string
}

func (b *SnapperBackend) findSnapshots(ctx context.Context, queryID string) ([]snapperMatch, error) {
	configs, err := b.getConfigs(ctx)
	if err != nil {
		return nil, err
	}

	var results []snapperMatch

	for cfgName, subvolPath := range configs {
		args := []string{"-c", cfgName, "--csvout", "list", "--columns", "number,userdata"}
		out, err := b.runWithSudo(ctx, args...)
		if err != nil {
			output.Warning("failed to list snapshots for config '%s': %v", cfgName, err)
			continue
		}

		r := csv.NewReader(strings.NewReader(string(out)))
		records, _ := r.ReadAll()

		for _, rec := range records {
			if len(rec) < 2 {
				continue
			}
			id := strings.TrimSpace(rec[0])
			if id == "number" || id == "#" {
				continue
			}
			userdata := strings.TrimSpace(rec[1])

			isMatch := false
			// Check for unified ID match in userdata
			if strings.Contains(userdata, fmt.Sprintf("shedman_id=%s", queryID)) {
				isMatch = true
			} else if id == queryID {
				isMatch = true
			}

			if isMatch {
				results = append(results, snapperMatch{
					ID:        id,
					Config:    cfgName,
					Subvolume: subvolPath,
				})
			}
		}
	}
	return results, nil
}

func (b *SnapperBackend) List(ctx context.Context, opts ListOptions) ([]Snapshot, error) {
	// Aggregation Map: UnifiedID -> Snapshot
	agg := make(map[string]*Snapshot)
	var others []Snapshot

	// Handle Remote Listing
	if opts.Target != nil {
		if b.cfg.Snapshot.RemoteStrategy == StrategyRestic {
			pwd := util.GetEnvOrPrompt("RESTIC_PASSWORD", "Enter Restic Repository Password: ")
			resticMgr := restic.NewManager(b.exec, pwd)
			rSnaps, err := resticMgr.List(ctx, opts.Target.Path)
			if err != nil {
				return nil, err
			}
			var results []Snapshot
			for _, r := range rSnaps {
				results = append(results, Snapshot{
					ID:        r.ID,
					Timestamp: r.Time,
					Tags:      r.Tags,
					Backend:   BackendRestic,
					// Description?
				})
			}
			return results, nil
		}
		if opts.Target.Path == "" {
			return nil, fmt.Errorf("remote target path is required")
		}

		// Rclone listing
		args := []string{"lsjson", "--dirs-only", opts.Target.Path}
		cmdArgs := util.GetPrivilegedRcloneCommand(args)

		out, err := b.exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...).Output()
		if err != nil {
			return nil, fmt.Errorf("rclone listing failed: %w", err)
		}

		var entries []struct {
			Name    string    `json:"Name"`
			ModTime time.Time `json:"ModTime"`
			IsDir   bool      `json:"IsDir"`
		}
		if err := json.Unmarshal(out, &entries); err != nil {
			return nil, fmt.Errorf("failed to parse rclone output: %w", err)
		}

		var results []Snapshot
		for _, e := range entries {
			if !e.IsDir {
				continue
			}

			// Parse timestamp from ID if possible (format: 20060102-150405)
			ts := e.ModTime
			if parsed, err := time.Parse("20060102-150405", e.Name); err == nil {
				ts = parsed
			}

			results = append(results, Snapshot{
				ID:          e.Name,
				Timestamp:   ts,
				Backend:     StrategyRclone,
				Description: "Remote Snapshot (Rclone)",
			})
		}
		return results, nil
	}

	configs, err := b.getConfigs(ctx)
	if err != nil {
		return nil, err
	}

	for cfgName := range configs {
		args := []string{"-c", cfgName, "--csvout", "--iso", "list", "--columns", "number,type,date,used-space,description,userdata"}
		out, err := b.runWithSudo(ctx, args...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to list snapshots for config '%s': %v\n", cfgName, err)
			continue
		}

		r := csv.NewReader(strings.NewReader(string(out)))
		records, _ := r.ReadAll()

		for _, record := range records {
			if len(record) < 6 {
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
			userdata := strings.TrimSpace(record[5])

			// Parse unified ID
			var unifiedID string
			if idx := strings.Index(userdata, "shedman_id="); idx != -1 {
				remainder := userdata[idx+len("shedman_id="):]
				if end := strings.IndexAny(remainder, " ,"); end != -1 {
					unifiedID = remainder[:end]
				} else {
					unifiedID = remainder
				}
			}

			ts, _ := time.Parse("2006-01-02 15:04:05", dateStr)
			size := util.ParseSize(sizeStr)

			if unifiedID != "" {
				if existing, ok := agg[unifiedID]; ok {
					existing.Size += size // Aggregate size
					// Description/Timestamp match presumed
				} else {
					agg[unifiedID] = &Snapshot{
						ID:          unifiedID,
						Description: desc,
						Type:        typ,
						Timestamp:   ts,
						Size:        size,
						Backend:     BackendSnapper,
					}
				}
			} else {
				// Legacy or Auto snapshot
				others = append(others, Snapshot{
					ID:          id, // Raw integer ID
					Description: fmt.Sprintf("[%s] %s", cfgName, desc),
					Type:        typ,
					Timestamp:   ts,
					Size:        size,
					Backend:     BackendSnapper,
				})
			}
		}
	}

	var final []Snapshot
	for _, s := range agg {
		final = append(final, *s)
	}
	final = append(final, others...)

	// Sort by timestamp descending
	sort.Slice(final, func(i, j int) bool {
		return final[i].Timestamp.After(final[j].Timestamp)
	})

	return final, nil
}

func (b *SnapperBackend) Delete(ctx context.Context, id string) error {
	matches, err := b.findSnapshots(ctx, id)
	if err != nil {
		return err
	}
	if len(matches) == 0 {
		return fmt.Errorf("snapshot %s not found", id)
	}

	for _, match := range matches {
		if _, err := b.runWithSudo(ctx, "-c", match.Config, "delete", match.ID); err != nil {
			return fmt.Errorf("failed to delete snapshot %s (config %s): %w", match.ID, match.Config, err)
		}
	}
	return nil
}

func (b *SnapperBackend) Restore(ctx context.Context, id string, opts RestoreOptions) error {
	if opts.DryRun {
		output.Info("Dry-run: %s %s %s", "sudo snapper", "rollback", id)
		return nil
	}
	_, err := b.runWithSudo(ctx, "rollback", id)
	return err
}
func (b *SnapperBackend) Prune(ctx context.Context, opts PruneOptions) error {
	snaps, err := b.List(ctx, ListOptions{})
	if err != nil {
		return err
	}

	if opts.KeepLast > 0 && len(snaps) > opts.KeepLast {
		toDelete := snaps[:len(snaps)-opts.KeepLast]
		for _, s := range toDelete {
			if s.ID == "0" {
				continue
			}
			if err := b.Delete(ctx, s.ID); err != nil {
				return fmt.Errorf("failed to prune snapshot %s: %w", s.ID, err)
			}
		}
	}

	return nil
}

func (b *SnapperBackend) Push(ctx context.Context, id string, target RemoteTarget, opts RemoteOptions) error {
	matches, err := b.findSnapshots(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to resolve snapshot %s: %w", id, err)
	}
	if len(matches) == 0 {
		return fmt.Errorf("snapshot %s not found (or no permission to list)", id)
	}

	output.Info("Deep-Pushing snapshot %s (Found %d subcomponents)", id, len(matches))

	if b.cfg.Snapshot.RemoteStrategy == StrategyRestic {
		pwd := util.GetEnvOrPrompt("RESTIC_PASSWORD", "Enter Restic Repository Password: ")
		resticMgr := restic.NewManager(b.exec, pwd)

		// Backup each subvolume component as a path in the repo
		for _, match := range matches {
			localPath := filepath.Join(match.Subvolume, ".snapshots", match.ID, "snapshot")
			if !strings.HasSuffix(localPath, "/") {
				localPath += "/"
			}

			// Tags: shedman, config:name, id:ID
			tags := []string{"shedman", fmt.Sprintf("config:%s", match.Config), fmt.Sprintf("id:%s", match.ID), fmt.Sprintf("unified:%s", id)}
			output.Info(">> Restic Backup: %s (ID %s) -> %s", match.Config, match.ID, target.Path)

			if err := resticMgr.Backup(ctx, target.Path, localPath, tags, os.Stdout, os.Stderr); err != nil {
				return fmt.Errorf("restic backup failed for %s: %w", match.Config, err)
			}
		}
		return nil
	}

	baseRemote := target.Path
	if !strings.HasSuffix(baseRemote, "/") && !strings.HasSuffix(baseRemote, ":") {
		baseRemote += "/"
	}
	unifiedBase := baseRemote + id

	for _, match := range matches {
		localPath := filepath.Join(match.Subvolume, ".snapshots", match.ID, "snapshot")
		if !strings.HasSuffix(localPath, "/") {
			localPath += "/"
		}

		remotePath := fmt.Sprintf("%s/%s", unifiedBase, match.Config)

		output.Info(">> Pushing component: %s (ID %s) -> %s", match.Config, match.ID, remotePath)

		args := []string{"sync", "-P", localPath, remotePath}
		if opts.Delete {
			args = append(args, "--delete")
		}
		if opts.Bandwidth > 0 {
			args = append(args, "--bwlimit", fmt.Sprintf("%dk", opts.Bandwidth))
		}

		cmdArgs := util.GetPrivilegedRcloneCommand(args)

		output.Info("Executing: %s", strings.Join(cmdArgs, " "))

		if opts.DryRun {
			continue
		}

		cmd := b.exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to sync component %s: %w", match.Config, err)
		}
	}
	return nil
}

func (b *SnapperBackend) Pull(ctx context.Context, id string, source RemoteTarget, opts RemoteOptions) error {
	// Check strategy
	if b.cfg.Snapshot.RemoteStrategy == StrategyRestic {
		pwd := util.GetEnvOrPrompt("RESTIC_PASSWORD", "Enter Restic Repository Password: ")
		resticMgr := restic.NewManager(b.exec, pwd)

		output.Info("Searching for snapshot %s in restic repo...", id)

		tags := []string{fmt.Sprintf("id:%s", id)}
		matches, err := resticMgr.FindByTags(ctx, source.Path, tags)
		if err != nil {
			return fmt.Errorf("failed to list remote snapshots: %w", err)
		}

		if len(matches) == 0 {
			return fmt.Errorf("snapshot %s not found in restic repo", id)
		}

		output.Info("Found %d components for snapshot %s. Restoring...", len(matches), id)

		for _, m := range matches {
			output.Info(">> Restoring component (Restic ID: %s)...", m.ShortID)

			if err := resticMgr.Restore(ctx, source.Path, m.ID, "/", os.Stdout, os.Stderr); err != nil {
				return fmt.Errorf("failed to restore %s: %w", m.ID, err)
			}
		}
		return nil
	}

	localPath := fmt.Sprintf("/.snapshots/%s/snapshot/", id)

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

	output.Info("Executing: rclone %s", strings.Join(args, " "))

	cmdArgs := util.GetPrivilegedRcloneCommand(args)

	output.Info("Executing: %s", strings.Join(cmdArgs, " "))

	if opts.DryRun {
		return nil
	}

	cmd := b.exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rclone sync failed: %w", err)
	}

	return nil
}

func (b *SnapperBackend) Diff(ctx context.Context, id1, id2 string) (DiffResult, error) {
	args := []string{"status", fmt.Sprintf("%s..%s", id1, id2)}

	out, err := b.runWithSudo(ctx, args...)
	if err != nil {
		return DiffResult{}, fmt.Errorf("snapper diff failed: %w", err)
	}

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
