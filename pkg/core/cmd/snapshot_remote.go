package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/snapshot"
)

var SnapshotRemoteCmd = &cobra.Command{
	Use:   "remote",
	Short: "Manage remote snapshots",
}

var SnapshotRemotePushCmd = &cobra.Command{
	Use:   "push <id> <target-name> [path]",
	Short: "Push a snapshot to a remote target",
	Args:  cobra.RangeArgs(2, 3), // id, target, optional path
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}

		path := ""
		if len(args) > 2 {
			path = args[2]
		}

		target := snapshot.RemoteTarget{
			Name: args[1],
			Path: path,
		}

		opts := snapshot.RemoteOptions{
			Compress:  snapshotRemoteCompress,
			Bandwidth: snapshotRemoteBandwidth,
			Delete:    snapshotRemoteDelete,
		}

		return RunSnapshotRemotePush(engine, args[0], target, opts, cmd.OutOrStdout())
	},
}

var SnapshotRemotePullCmd = &cobra.Command{
	Use:   "pull <id> <source-name> [path]",
	Short: "Pull a snapshot from a remote source",
	Args:  cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}

		path := ""
		if len(args) > 2 {
			path = args[2]
		}

		source := snapshot.RemoteTarget{
			Name: args[1],
			Path: path,
		}

		opts := snapshot.RemoteOptions{
			Compress:  snapshotRemoteCompress,
			Bandwidth: snapshotRemoteBandwidth,
			Delete:    snapshotRemoteDelete,
		}

		return RunSnapshotRemotePull(engine, args[0], source, opts, cmd.OutOrStdout())
	},
}

func init() {
	SnapshotRemoteCmd.AddCommand(SnapshotRemotePushCmd)
	SnapshotRemoteCmd.AddCommand(SnapshotRemotePullCmd)

	// Flags for Push
	SnapshotRemotePushCmd.Flags().BoolVar(&snapshotRemoteCompress, "compress", true, "Enable compression")
	SnapshotRemotePushCmd.Flags().BoolVar(&snapshotRemoteDelete, "delete", false, "Delete extraneous files on destination")
	SnapshotRemotePushCmd.Flags().IntVar(&snapshotRemoteBandwidth, "bwlimit", 0, "Bandwidth limit in KB/s")

	// Flags for Pull
	SnapshotRemotePullCmd.Flags().BoolVar(&snapshotRemoteCompress, "compress", true, "Enable compression")
	SnapshotRemotePullCmd.Flags().BoolVar(&snapshotRemoteDelete, "delete", false, "Delete extraneous files on local (cautious usage)")
	SnapshotRemotePullCmd.Flags().IntVar(&snapshotRemoteBandwidth, "bwlimit", 0, "Bandwidth limit in KB/s")
}

var (
	snapshotRemoteCompress  bool
	snapshotRemoteDelete    bool
	snapshotRemoteBandwidth int
)

// Logic

func RunSnapshotRemotePush(engine *core.Engine, id string, target snapshot.RemoteTarget, opts snapshot.RemoteOptions, w io.Writer) error {
	mgr := engine.GetSnapshotManager()
	if mgr == nil {
		return fmt.Errorf("snapshot manager not available")
	}

	fmt.Fprintf(w, "Pushing snapshot %s to %s...\n", id, target.Name)
	if err := mgr.Push(id, target, opts); err != nil {
		return fmt.Errorf("push failed: %w", err)
	}
	fmt.Fprintln(w, "Push successful.")
	return nil
}

func RunSnapshotRemotePull(engine *core.Engine, id string, source snapshot.RemoteTarget, opts snapshot.RemoteOptions, w io.Writer) error {
	mgr := engine.GetSnapshotManager()
	if mgr == nil {
		return fmt.Errorf("snapshot manager not available")
	}

	fmt.Fprintf(w, "Pulling snapshot %s from %s...\n", id, source.Name)
	if err := mgr.Pull(id, source, opts); err != nil {
		return fmt.Errorf("pull failed: %w", err)
	}
	fmt.Fprintln(w, "Pull successful.")
	return nil
}
