package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/pkg/snapshot"
)

// SnapshotPushCmd pushes a snapshot to a remote target.
var SnapshotPushCmd = &cobra.Command{
	Use:   "push <id> [target-name] [path]",
	Short: "Push a snapshot to a remote target",
	Args:  cobra.RangeArgs(1, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}

		targetName := ""
		if len(args) > 1 {
			targetName = args[1]
		}

		path := ""
		if len(args) > 2 {
			path = args[2]
		}

		cfg := engine.GetConfig()
		if targetName == "" {
			if cfg.Snapshot.DefaultRemote != "" {
				targetName = cfg.Snapshot.DefaultRemote
			} else {
				if len(cfg.Snapshot.Remotes) == 1 {
					for k := range cfg.Snapshot.Remotes {
						targetName = k
						break
					}
				} else {
					return fmt.Errorf("no target specified and no default_remote configured (remotes available: %d)", len(cfg.Snapshot.Remotes))
				}
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Using default remote: %s\n", targetName)
		}

		target := snapshot.RemoteTarget{
			Name: targetName,
			Path: path,
		}

		opts := snapshot.RemoteOptions{
			Compress:  snapshotRemoteCompress,
			Bandwidth: snapshotRemoteBandwidth,
			Delete:    snapshotRemoteDelete,
		}

		return RunSnapshotRemotePush(cmd.Context(), engine, args[0], target, opts, cmd.OutOrStdout())
	},
}

// SnapshotPullCmd pulls a snapshot from a remote target.
var SnapshotPullCmd = &cobra.Command{
	Use:   "pull <id> [source-name] [path]",
	Short: "Pull a snapshot from a remote source",
	Args:  cobra.RangeArgs(1, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}

		sourceName := ""
		if len(args) > 1 {
			sourceName = args[1]
		}

		path := ""
		if len(args) > 2 {
			path = args[2]
		}

		cfg := engine.GetConfig()
		if sourceName == "" {
			if cfg.Snapshot.DefaultRemote != "" {
				sourceName = cfg.Snapshot.DefaultRemote
			} else {
				if len(cfg.Snapshot.Remotes) == 1 {
					for k := range cfg.Snapshot.Remotes {
						sourceName = k
						break
					}
				} else {
					return fmt.Errorf("no source specified and no default_remote configured")
				}
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Using default remote: %s\n", sourceName)
		}

		source := snapshot.RemoteTarget{
			Name: sourceName,
			Path: path,
		}

		opts := snapshot.RemoteOptions{
			Compress:  snapshotRemoteCompress,
			Bandwidth: snapshotRemoteBandwidth,
			Delete:    snapshotRemoteDelete,
		}

		return RunSnapshotRemotePull(cmd.Context(), engine, args[0], source, opts, cmd.OutOrStdout())
	},
}

func init() {
	SnapshotCmd.AddCommand(SnapshotPushCmd)
	SnapshotCmd.AddCommand(SnapshotPullCmd)

	SnapshotPushCmd.Flags().BoolVar(&snapshotRemoteDelete, "delete", false, "Delete extraneous files on destination")
	SnapshotPushCmd.Flags().BoolVar(&snapshotRemoteCompress, "compress", false, "Enable compression during transfer")
	SnapshotPushCmd.Flags().BoolVar(&snapshotRemoteEncrypt, "encrypt", false, "Require encrypted (restic) transfer")
	SnapshotPushCmd.Flags().IntVar(&snapshotRemoteBandwidth, "bwlimit", 0, "Bandwidth limit in KB/s")

	SnapshotPullCmd.Flags().BoolVar(&snapshotRemoteDelete, "delete", false, "Delete extraneous files on local (cautious usage)")
	SnapshotPullCmd.Flags().BoolVar(&snapshotRemoteCompress, "compress", false, "Enable compression during transfer")
	SnapshotPullCmd.Flags().BoolVar(&snapshotRemoteDecrypt, "decrypt", false, "Require encrypted (restic) transfer")
	SnapshotPullCmd.Flags().IntVar(&snapshotRemoteBandwidth, "bwlimit", 0, "Bandwidth limit in KB/s")
}
