package cmd

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/pkg/core"
)

// SnapshotDiffCmd is the command to show differences between snapshots
var SnapshotDiffCmd = &cobra.Command{
	Use:   "diff <id1> <id2>",
	Short: "Show differences between two snapshots",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}
		return RunSnapshotDiff(engine, args[0], args[1], cmd.OutOrStdout())
	},
}

func RunSnapshotDiff(engine *core.Engine, id1, id2 string, w io.Writer) error {
	mgr := engine.GetSnapshotManager()
	if mgr == nil {
		return fmt.Errorf("snapshot manager not available")
	}

	diff, err := mgr.Diff(id1, id2)
	if err != nil {
		return fmt.Errorf("diff failed: %w", err)
	}

	if len(diff.Added) == 0 && len(diff.Removed) == 0 && len(diff.Modified) == 0 {
		_, _ = fmt.Fprintln(w, "No differences found.")

		return nil
	}

	_, _ = fmt.Fprintf(w, "Diff between %s and %s:\n", id1, id2)

	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)

	if len(diff.Added) > 0 {
		_, _ = fmt.Fprintln(tw, "\n[ADDED]:")

		for _, item := range diff.Added {
			_, _ = fmt.Fprintf(tw, "  + %s\n", item)

		}
	}

	if len(diff.Removed) > 0 {
		_, _ = fmt.Fprintln(tw, "\n[REMOVED]:")
		for _, item := range diff.Removed {
			_, _ = fmt.Fprintf(tw, "  - %s\n", item)
		}
	}

	if len(diff.Modified) > 0 {
		_, _ = fmt.Fprintln(tw, "\n[MODIFIED]:")
		for _, item := range diff.Modified {
			_, _ = fmt.Fprintf(tw, "  * %s\n", item)
		}
	}

	_ = tw.Flush()

	return nil
}
