package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/disk"
	"github.com/theshedman/shedman/internal/util"
	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/executor"
	"github.com/theshedman/shedman/pkg/snapshot"
	"github.com/theshedman/shedman/pkg/snapshot/restic"
)

var SnapshotDiskCmd = &cobra.Command{
	Use:   "disk",
	Short: "Manage snapshots on local disk or USB",
}

var SnapshotDiskSaveCmd = &cobra.Command{
	Use:   "save <id> <target-path-or-device>",
	Short: "Save a snapshot to a local disk or USB drive",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}
		return RunSnapshotDiskSave(cmd.Context(), engine, args[0], args[1], cmd.OutOrStdout())
	},
}

var SnapshotDiskRestoreCmd = &cobra.Command{
	Use:   "restore <target-path-or-device> <id>",
	Short: "Restore a snapshot from a local disk or USB drive",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}
		return RunSnapshotDiskRestore(cmd.Context(), engine, args[0], args[1], cmd.OutOrStdout())
	},
}

var SnapshotDiskListCmd = &cobra.Command{
	Use:   "list <target-path-or-device>",
	Short: "List snapshots on a local disk or USB drive",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}
		return RunSnapshotDiskList(cmd.Context(), engine, args[0], cmd.OutOrStdout())
	},
}

func init() {
	SnapshotDiskCmd.AddCommand(SnapshotDiskSaveCmd)
	SnapshotDiskCmd.AddCommand(SnapshotDiskRestoreCmd)
	SnapshotDiskCmd.AddCommand(SnapshotDiskListCmd)

	SnapshotDiskSaveCmd.Flags().BoolVar(&snapshotDiskFormat, "format", false, "Format target as ext4 first")
	SnapshotDiskSaveCmd.Flags().BoolVar(&snapshotDiskCompress, "compress", false, "Enable compression during transfer")
	SnapshotDiskSaveCmd.Flags().BoolVar(&snapshotDiskVerify, "verify", false, "Verify after write")
}

var (
	snapshotDiskFormat   bool
	snapshotDiskCompress bool
	snapshotDiskVerify   bool
)

// resolveDiskTarget handles mounting if necessary
func resolveDiskTarget(target string, format bool) (string, func(), error) {
	cleanup := func() {}

	// If it's a block device
	if strings.HasPrefix(target, "/dev/") {
		diskMgr := disk.NewManager(&executor.RealExecutor{})

		if format {
			if err := diskMgr.FormatExt4(target); err != nil {
				return "", cleanup, fmt.Errorf("failed to format device: %w", err)
			}
		}

		if err := diskMgr.CheckSafeguards(target); err != nil {
			return "", cleanup, fmt.Errorf("safety check failed for %s: %w", target, err)
		}

		mountPoint, unmount, err := diskMgr.Mount(target)
		if err != nil {
			return "", cleanup, fmt.Errorf("failed to mount device: %w", err)
		}

		// Setup signal handling to unmount on Ctrl+C
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-c
			fmt.Println("\nInterrupted. Unmounting...")
			unmount()
			os.Exit(1)
		}()

		cleanup = func() {
			signal.Stop(c)
			unmount()
		}

		repoPath := filepath.Join(mountPoint, "shedman-backup")
		if err := os.MkdirAll(repoPath, 0755); err != nil {
			// Try with sudo if permission denied
			if mkErr := (&executor.RealExecutor{}).Command("sudo", "mkdir", "-p", repoPath).Run(); mkErr != nil {
				cleanup()
				return "", nil, fmt.Errorf("failed to create backup directory %s: %w", repoPath, err)
			}
		}
		return repoPath, cleanup, nil
	}

	// Just a path
	repoPath := filepath.Join(target, "shedman-backup")
	return repoPath, cleanup, nil
}

func RunSnapshotDiskSave(ctx context.Context, engine *core.Engine, id string, targetDevice string, w io.Writer) error {
	repoPath, cleanup, err := resolveDiskTarget(targetDevice, snapshotDiskFormat)
	if err != nil {
		return err
	}
	defer cleanup()

	mgr := engine.GetSnapshotManager()
	if mgr == nil {
		return fmt.Errorf("snapshot manager not available")
	}

	_, _ = fmt.Fprintf(w, "Saving snapshot %s to %s...\n", id, repoPath)

	// Ensure repository is initialized
	configPath := filepath.Join(repoPath, "config")
	diskPwd := os.Getenv("RESTIC_PASSWORD")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		_, _ = fmt.Fprintln(w, "Repository not initialized. Initializing...")

		// Initialize new restic repository for local disk
		// We need to establish a password here which will be used by the Push operation later.

		if diskPwd == "" {
			fmt.Println("Initializing new disk repository.")
			fmt.Print("Enter password (leave empty for 'shedman'): ")
			var input string
			_, _ = fmt.Scanln(&input)
			if input == "" {
				diskPwd = "shedman"
			} else {
				diskPwd = input
			}
			_ = os.Setenv("RESTIC_PASSWORD", diskPwd)
		}

		exec := &executor.RealExecutor{}
		resticMgr := restic.NewManager(exec, diskPwd)

		if err := resticMgr.Init(ctx, repoPath, w); err != nil {
			return fmt.Errorf("failed to init disk repo: %w", err)
		}
	}

	target := snapshot.RemoteTarget{
		Name: snapshot.StrategyLocal, // This triggers our logic? backend_snapper.go doesn't check type.
		Path: repoPath,
		Type: snapshot.StrategyLocal,
	}

	opts := snapshot.RemoteOptions{
		Compress: snapshotDiskCompress,
	}

	if err := mgr.Push(ctx, id, target, opts); err != nil {
		return err
	}

	if snapshotDiskVerify {
		cfg := engine.GetConfig()
		if cfg != nil && cfg.Snapshot.RemoteStrategy == snapshot.StrategyRestic {
			if diskPwd == "" {
				diskPwd = util.GetEnvOrPrompt("RESTIC_PASSWORD", "Enter Restic Repository Password: ")
			}
			resticMgr := restic.NewManager(&executor.RealExecutor{}, diskPwd)
			if err := resticMgr.Check(ctx, repoPath, w); err != nil {
				return fmt.Errorf("verification failed: %w", err)
			}
		} else {
			_, _ = fmt.Fprintln(w, "Verification not supported for current remote strategy.")
		}
	}

	_, _ = fmt.Fprintln(w, "Snapshot saved successfully.")
	return nil
}

func RunSnapshotDiskRestore(ctx context.Context, engine *core.Engine, targetDevice string, id string, w io.Writer) error {
	repoPath, cleanup, err := resolveDiskTarget(targetDevice, false)
	if err != nil {
		return err
	}
	defer cleanup()

	mgr := engine.GetSnapshotManager()
	if mgr == nil {
		return fmt.Errorf("snapshot manager not available")
	}

	_, _ = fmt.Fprintf(w, "Restoring %s from %s...\n", id, repoPath)

	source := snapshot.RemoteTarget{
		Name: "disk",
		Path: repoPath,
		Type: snapshot.StrategyLocal,
	}

	opts := snapshot.RemoteOptions{}

	if err := mgr.Pull(ctx, id, source, opts); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(w, "Snapshot restored successfully.")
	return nil
}

func RunSnapshotDiskList(ctx context.Context, engine *core.Engine, targetDevice string, w io.Writer) error {
	repoPath, cleanup, err := resolveDiskTarget(targetDevice, false)
	if err != nil {
		return err
	}
	defer cleanup()

	mgr := engine.GetSnapshotManager()
	if mgr == nil {
		return fmt.Errorf("snapshot manager not available")
	}

	// We use List with new Target option
	target := &snapshot.RemoteTarget{
		Path: repoPath,
		Type: snapshot.StrategyLocal,
	}

	opts := snapshot.ListOptions{
		Target: target,
	}

	snaps, err := mgr.List(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to list snapshots on disk: %w", err)
	}

	if len(snaps) == 0 {
		_, _ = fmt.Fprintln(w, "No snapshots found on disk.")
		return nil
	}

	_, _ = fmt.Fprintln(w, "ID\tTIMESTAMP\t\tDESCRIPTION")
	for _, s := range snaps {
		id := s.ID
		if len(id) > 8 {
			id = id[:8]
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", id, s.Timestamp.Format("2006-01-02 15:04"), s.Tags)
	}

	return nil
}
