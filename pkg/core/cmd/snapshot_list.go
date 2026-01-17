package cmd

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"
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

		return RunSnapshotList(engine, opts, cmd.OutOrStdout())
	},
}

func RunSnapshotList(engine *core.Engine, opts snapshot.ListOptions, w io.Writer) error {
	mgr := engine.GetSnapshotManager()
	if mgr == nil {
		return fmt.Errorf("snapshot manager not available (check config or installed tools)")
	}

	snapshots, err := mgr.List(opts)
	if err != nil {
		return fmt.Errorf("failed to list snapshots: %w", err)
	}

	if len(snapshots) == 0 {
		_, _ = fmt.Fprintln(w, "No snapshots found.")

		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	fmt.Fprintln(tw, "ID\tDATE\tTYPE\tBACKEND\tDESCRIPTION")

	for _, snap := range snapshots {
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			snap.ID,
			snap.Date.Format("2006-01-02 15:04:05"),
			snap.Type,
			snap.Backend,
			snap.Description)

	}
	_ = tw.Flush()

	return nil
}

func init() {
	SnapshotListCmd.Flags().BoolVar(&snapshotListRemote, "remote", false, "List remote snapshots (if configured)")
}
