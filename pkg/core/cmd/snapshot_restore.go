package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/snapshot"
)

// SnapshotRestoreCmd is the command to restore a snapshot
var SnapshotRestoreCmd = &cobra.Command{
	Use:   "restore <id>",
	Short: "Restore a snapshot",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return fmt.Errorf("failed to initialize engine: %w", err)
		}

		opts := snapshot.RestoreOptions{
			PackagesOnly: snapshotRestorePackagesOnly,
			ConfigsOnly:  snapshotRestoreConfigsOnly,
			HomeOnly:     snapshotRestoreHomeOnly,
		}

		return RunSnapshotRestore(engine, args, opts, cmd.OutOrStdout())
	},
}

var (
	snapshotRestorePackagesOnly bool
	snapshotRestoreConfigsOnly  bool
	snapshotRestoreHomeOnly     bool
)

func RunSnapshotRestore(engine *core.Engine, args []string, opts snapshot.RestoreOptions, w io.Writer) error {
	mgr := engine.GetSnapshotManager()
	if mgr == nil {
		return fmt.Errorf("snapshot manager not available")
	}

	id := args[0]

	_, _ = fmt.Fprintf(w, "Restoring snapshot %s...\n", id)

	if err := mgr.Restore(id, opts); err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}

	_, _ = fmt.Fprintln(w, "Snapshot restored successfully.")

	return nil
}

func init() {
	SnapshotRestoreCmd.Flags().BoolVar(&snapshotRestorePackagesOnly, "packages-only", false, "Restore only packages")
	SnapshotRestoreCmd.Flags().BoolVar(&snapshotRestoreConfigsOnly, "configs-only", false, "Restore only configurations")
	SnapshotRestoreCmd.Flags().BoolVar(&snapshotRestoreHomeOnly, "home-only", false, "Restore only home directory files")
}
