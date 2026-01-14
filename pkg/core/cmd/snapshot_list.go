package cmd

import (
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

		return RunSnapshotList(engine, opts, cmd.OutOrStdout())
	},
}

// RunSnapshotList executes the snapshot listing logic
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
		fmt.Fprintln(w, "No snapshots found.")
		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	fmt.Fprintln(tw, "ID\tTIMESTAMP\tBACKEND\tSIZE\tDESCRIPTION")

	for _, snap := range snapshots {
		ts := snap.Timestamp.Format(time.RFC3339)		
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			snap.ID,
			ts,
			snap.Backend,
			util.FormatSize(snap.Size),
			snap.Description)
	}
	tw.Flush()

	return nil
}

func init() {
	SnapshotListCmd.Flags().BoolVar(&snapshotListRemote, "remote", false, "List remote snapshots (if configured)")
}
