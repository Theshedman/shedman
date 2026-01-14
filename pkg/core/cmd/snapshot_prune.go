package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/snapshot"
)

var SnapshotPruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Prune old snapshots",
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}

		// Parse flags for options (keep-last, keep-scheduled, older-than)
		// For now simple default options or parsed from flags
		opts := snapshot.PruneOptions{
			KeepLast: 5, // Default example
		}

		return RunSnapshotPrune(engine, opts, cmd.OutOrStdout())
	},
}

func RunSnapshotPrune(engine *core.Engine, opts snapshot.PruneOptions, w io.Writer) error {
	mgr := engine.GetSnapshotManager()
	if mgr == nil {
		return fmt.Errorf("snapshot manager not available")
	}

	fmt.Fprintln(w, "Pruning snapshots...")
	if err := mgr.Prune(opts); err != nil {
		return fmt.Errorf("prune failed: %w", err)
	}
	fmt.Fprintln(w, "Prune completed.")
	return nil
}

func init() {
	// Add flags: --keep-last, --older-than, etc.
	SnapshotCmd.AddCommand(SnapshotPruneCmd)
}
