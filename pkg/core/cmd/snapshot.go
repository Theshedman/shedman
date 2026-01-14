package cmd

import (
	"github.com/spf13/cobra"
)

// SnapshotCmd is the root command for snapshot operations
var SnapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Manage system snapshots",
	Long:  `Create, list, restore, and manage system snapshots using various backends (Snapper, Timeshift, Rsync).`,
}

func init() {
	SnapshotCmd.AddCommand(SnapshotCreateCmd)
	SnapshotCmd.AddCommand(SnapshotListCmd)
	SnapshotCmd.AddCommand(SnapshotRestoreCmd)
	SnapshotCmd.AddCommand(SnapshotDeleteCmd)
	SnapshotCmd.AddCommand(SnapshotScheduleCmd)
	SnapshotCmd.AddCommand(SnapshotKeyCmd)
	SnapshotCmd.AddCommand(SnapshotDiffCmd)
	SnapshotCmd.AddCommand(SnapshotRemoteCmd)
	SnapshotCmd.AddCommand(SnapshotMigrateCmd)
	// SnapshotPruneCmd registered in its own init()
}
