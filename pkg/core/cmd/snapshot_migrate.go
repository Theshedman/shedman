package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/pkg/core"
)

var SnapshotMigrateCmd = &cobra.Command{
	Use:   "migrate <target-backend>",
	Short: "Migrate snapshots to another backend",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}
		return RunSnapshotMigrate(engine, args[0], cmd.OutOrStdout())
	},
}

func RunSnapshotMigrate(engine *core.Engine, targetBackend string, w io.Writer) error {
	mgr := engine.GetSnapshotManager()
	if mgr == nil {
		return fmt.Errorf("snapshot manager not available")
	}

	currentBackend := mgr.GetBackendName()
	if currentBackend == targetBackend {
		_, _ = fmt.Fprintf(w, "Already on backend '%s'. Nothing to do.\n", targetBackend)

		return nil
	}

	_, _ = fmt.Fprintf(w, "Migrating snapshots from '%s' to '%s'...\n", currentBackend, targetBackend)

	// Validate Target
	if targetBackend != "rsync" && targetBackend != "timeshift" && targetBackend != "snapper" {
		return fmt.Errorf("unknown target backend '%s'", targetBackend)
	}

	if currentBackend == "rsync" && targetBackend == "rsync" {
		_, _ = fmt.Fprintln(w, "To migrate rsync storage location, move the directory manually and update config.")

		return nil
	}

	return fmt.Errorf("automatic migration from '%s' to '%s' is not currently supported due to filesystem differences. Please migrate data manually", currentBackend, targetBackend)

}

func init() {
	SnapshotCmd.AddCommand(SnapshotMigrateCmd)
}
