package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/executor"
	"github.com/theshedman/shedman/pkg/snapshot"
)

var (
	snapshotCreateIncludeHome bool
	snapshotCreateType        string
	snapshotCreateTags        []string
	snapshotCreateTargets     []string
)

// SnapshotCreateCmd is the command to create a new snapshot
var SnapshotCreateCmd = &cobra.Command{
	Use:   "create [description]",
	Short: "Create a new snapshot",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return fmt.Errorf("failed to initialize engine: %w", err)
		}

		opts := snapshot.CreateOptions{
			Type:          snapshotCreateType,
			IncludeHome:   snapshotCreateIncludeHome,
			Tags:          snapshotCreateTags,
			TargetConfigs: snapshotCreateTargets,
		}

		return RunSnapshotCreate(engine, args, opts, cmd.OutOrStdout())
	},
}

func RunSnapshotCreate(engine *core.Engine, args []string, opts snapshot.CreateOptions, w io.Writer) error {
	mgr := engine.GetSnapshotManager()
	if mgr == nil {
		return fmt.Errorf("snapshot manager not available (check config or installed tools)")
	}

	var description string
	if len(args) > 0 {
		description = args[0]
	}

	if description == "" {
		description = fmt.Sprintf("Manual snapshot %s", opts.Type)
	}

	// Hooks
	var hooks config.SnapshotHooksConfig // Default empty
	if cfg := engine.GetConfig(); cfg != nil {
		hooks = cfg.Snapshot.Hooks
	}

	if hook := hooks.PreCreate; hook != "" {
		_, _ = fmt.Fprintf(w, "Executing pre-snapshot hook: %s\n", hook)

		// Run via shell to support piping/redirection
		cmd := (&executor.RealExecutor{}).Command("sh", "-c", hook)

		cmd.Stdout = w
		cmd.Stderr = w
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("pre-snapshot hook failed: %w", err)
		}
	}

	snap, err := mgr.Create(description, opts)
	if err != nil {
		return err
	}

	if hook := hooks.PostCreate; hook != "" {
		_, _ = fmt.Fprintf(w, "Executing post-snapshot hook: %s\n", hook)

		cmd := (&executor.RealExecutor{}).Command("sh", "-c", hook)

		cmd.Stdout = w
		cmd.Stderr = w
		if err := cmd.Run(); err != nil {
			_, _ = fmt.Fprintf(w, "Warning: post-snapshot hook failed: %v\n", err)

		}
	}

	_, _ = fmt.Fprintf(w, "Snapshot created successfully.\nID: %s\nBackend: %s\n", snap.ID, snap.Backend)

	return nil
}

func init() {
	SnapshotCreateCmd.Flags().StringVarP(&snapshotCreateType, "type", "t", "single", "Snapshot type (single, pre, post, ondemand)")
	SnapshotCreateCmd.Flags().BoolVar(&snapshotCreateIncludeHome, "include-home", false, "Include home directory")
	SnapshotCreateCmd.Flags().StringSliceVar(&snapshotCreateTags, "tags", nil, "Tags for the snapshot")
	SnapshotCreateCmd.Flags().StringSliceVar(&snapshotCreateTargets, "target", nil, "Target configs/subvolumes (for snapper)")
}
