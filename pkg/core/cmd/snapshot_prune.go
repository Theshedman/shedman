package cmd

import (
	"fmt"
	"io"
	"time"

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

		// Parse duration if provided
		var duration time.Duration
		if snapshotPruneOlderThan != "" {
			d := snapshotPruneOlderThan
			if len(d) > 1 && d[len(d)-1] == 'd' {
				days := 0
				fmt.Sscanf(d, "%dd", &days)
				duration = time.Duration(days) * 24 * time.Hour
			} else {
				var err error
				duration, err = time.ParseDuration(d)
				if err != nil {
					return fmt.Errorf("invalid duration format for --older-than: %v", err)
				}
			}
		}

		opts := snapshot.PruneOptions{
			KeepLast:      snapshotPruneKeepLast,
			KeepScheduled: snapshotPruneKeepScheduled,
			OlderThan:     duration,
		}

		return RunSnapshotPrune(engine, opts, cmd.OutOrStdout())
	},
}

var (
	snapshotPruneKeepLast      int
	snapshotPruneKeepScheduled int
	snapshotPruneOlderThan     string
)

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
	SnapshotPruneCmd.Flags().IntVar(&snapshotPruneKeepLast, "keep-last", 0, "Number of recent snapshots to keep")
	SnapshotPruneCmd.Flags().IntVar(&snapshotPruneKeepScheduled, "keep-scheduled", 0, "Number of scheduled snapshots to keep")
	SnapshotPruneCmd.Flags().StringVar(&snapshotPruneOlderThan, "older-than", "", "Delete snapshots older than duration (e.g. 24h, 7d)")
	SnapshotCmd.AddCommand(SnapshotPruneCmd)
}
