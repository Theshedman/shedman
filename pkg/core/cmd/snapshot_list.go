package cmd

import (
	"context"
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/util"
	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/snapshot"
)

var (
	snapshotListRemote bool
)

// SnapshotListCmd is the command to list snapshots
var SnapshotListCmd = &cobra.Command{
	Use:   "list",
	Short: "List snapshots",
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return fmt.Errorf("failed to initialize engine: %w", err)
		}

		opts := snapshot.ListOptions{}
		if snapshotListRemote {
			cfg := engine.GetConfig()
			if cfg == nil {
				return fmt.Errorf("snapshot config not available")
			}

			targetName := cfg.Snapshot.DefaultRemote
			if targetName == "" {
				if len(cfg.Snapshot.Remotes) == 1 {
					for k := range cfg.Snapshot.Remotes {
						targetName = k
						break
					}
				} else {
					return fmt.Errorf("no remote specified and no default_remote configured")
				}
			}

			remote := cfg.Snapshot.Remotes[targetName]
			target := snapshot.RemoteTarget{
				Name: targetName,
				Type: remote.Type,
				Path: remote.Path,
			}
			if target.Path == "" {
				target.Path = targetName + ":"
			}

			opts.Remote = true
			opts.Target = &target
		}

		return RunSnapshotList(cmd.Context(), engine, opts, cmd.OutOrStdout())
	},
}

func RunSnapshotList(ctx context.Context, engine *core.Engine, opts snapshot.ListOptions, w io.Writer) error {
	mgr := engine.GetSnapshotManager()
	if mgr == nil {
		return fmt.Errorf("snapshot manager not available (check config or installed tools)")
	}

	snapshots, err := mgr.List(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to list snapshots: %w", err)
	}

	if len(snapshots) == 0 {
		_, _ = fmt.Fprintln(w, "No snapshots found.")

		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	_, _ = fmt.Fprintln(tw, "ID\tTIMESTAMP\tBACKEND\tSIZE\tDESCRIPTION")

	for _, snap := range snapshots {
		ts := snap.Timestamp.Format(time.RFC3339)
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			snap.ID,
			ts,
			snap.Backend,
			util.FormatSize(snap.Size),
			snap.Description)

	}
	_ = tw.Flush()

	return nil
}

func init() {
	SnapshotListCmd.Flags().BoolVar(&snapshotListRemote, "remote", false, "List remote snapshots (if configured)")
}
