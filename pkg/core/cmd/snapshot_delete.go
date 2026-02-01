package cmd

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/snapshot"
)

// SnapshotDeleteCmd is the command to delete a snapshot
var SnapshotDeleteCmd = &cobra.Command{
	Use:   "delete [id]",
	Short: "Delete a snapshot",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return fmt.Errorf("failed to initialize engine: %w", err)
		}

		var olderThan time.Duration
		if snapshotDeleteOlderThanRaw != "" {
			d, err := parseDurationWithDays(snapshotDeleteOlderThanRaw)
			if err != nil {
				return err
			}
			olderThan = d
		}

		opts := SnapshotDeleteOptions{
			OlderThan: olderThan,
		}

		return RunSnapshotDelete(cmd.Context(), engine, args, opts, cmd.OutOrStdout())
	},
}

type SnapshotDeleteOptions struct {
	OlderThan time.Duration
}

func RunSnapshotDelete(ctx context.Context, engine *core.Engine, args []string, opts SnapshotDeleteOptions, w io.Writer) error {
	mgr := engine.GetSnapshotManager()
	if mgr == nil {
		return fmt.Errorf("snapshot manager not available")
	}

	if opts.OlderThan > 0 {
		_, _ = fmt.Fprintln(w, "Deleting snapshots older than threshold...")
		if err := mgr.Prune(ctx, snapshot.PruneOptions{OlderThan: opts.OlderThan}); err != nil {
			return fmt.Errorf("prune failed: %w", err)
		}
		_, _ = fmt.Fprintln(w, "Prune completed.")
		return nil
	}

	if len(args) == 0 {
		return fmt.Errorf("snapshot id is required unless --older-than is set")
	}

	id := args[0]

	_, _ = fmt.Fprintf(w, "Deleting snapshot %s...\n", id)

	if err := mgr.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}

	_, _ = fmt.Fprintln(w, "Snapshot deleted successfully.")

	return nil
}

var snapshotDeleteOlderThanRaw string

func parseDurationWithDays(value string) (time.Duration, error) {
	if len(value) > 1 && value[len(value)-1] == 'd' {
		days := 0
		if _, err := fmt.Sscanf(value, "%dd", &days); err != nil {
			return 0, fmt.Errorf("invalid duration format for --older-than: %v", err)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}

	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid duration format for --older-than: %v", err)
	}
	return d, nil
}

func init() {
	SnapshotDeleteCmd.Flags().StringVar(&snapshotDeleteOlderThanRaw, "older-than", "", "Delete snapshots older than duration (e.g. 24h, 7d)")
}
