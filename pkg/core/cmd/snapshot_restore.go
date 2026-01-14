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
			// Add flags here if needed (e.g. packages only)
		}

		return RunSnapshotRestore(engine, args, opts, cmd.OutOrStdout())
	},
}

// RunSnapshotRestore executes the snapshot restore logic
func RunSnapshotRestore(engine *core.Engine, args []string, opts snapshot.RestoreOptions, w io.Writer) error {
	mgr := engine.GetSnapshotManager()
	if mgr == nil {
		return fmt.Errorf("snapshot manager not available")
	}

	id := args[0]

	fmt.Fprintf(w, "Restoring snapshot %s...\n", id)
	if err := mgr.Restore(id, opts); err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}

	fmt.Fprintln(w, "Snapshot restored successfully.")
	return nil
}

func init() {
	// Add flags
}
