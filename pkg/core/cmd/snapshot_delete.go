package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/pkg/core"
)

// SnapshotDeleteCmd is the command to delete a snapshot
var SnapshotDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a snapshot",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return fmt.Errorf("failed to initialize engine: %w", err)
		}

		return RunSnapshotDelete(engine, args, cmd.OutOrStdout())
	},
}

func RunSnapshotDelete(engine *core.Engine, args []string, w io.Writer) error {
	mgr := engine.GetSnapshotManager()
	if mgr == nil {
		return fmt.Errorf("snapshot manager not available")
	}

	id := args[0]

	_, _ = fmt.Fprintf(w, "Deleting snapshot %s...\n", id)

	if err := mgr.Delete(id); err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}

	_, _ = fmt.Fprintln(w, "Snapshot deleted successfully.")

	return nil
}
