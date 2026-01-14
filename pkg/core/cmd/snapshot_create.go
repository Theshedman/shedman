package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/snapshot"
)

var (
	snapshotCreateLimit       int
	snapshotCreateIncludeHome bool
	snapshotCreateType        string
	snapshotCreateTags        []string
)

// SnapshotCreateCmd is the command to create a new snapshot
var SnapshotCreateCmd = &cobra.Command{
	Use:   "create [description]",
	Short: "Create a new snapshot",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return fmt.Errorf("failed to initialize engine: %w", err)
		}

		opts := snapshot.CreateOptions{
			Type:        snapshotCreateType,
			IncludeHome: snapshotCreateIncludeHome,
			Tags:        snapshotCreateTags,
		}

		return RunSnapshotCreate(engine, args, opts, cmd.OutOrStdout())
	},
}

func RunSnapshotCreate(engine *core.Engine, args []string, opts snapshot.CreateOptions, w io.Writer) error {
	mgr := engine.GetSnapshotManager()
	if mgr == nil {
		return fmt.Errorf("snapshot manager not available (check config or installed tools)")
	}

	var description string
	if len(args) > 0 {
		description = args[0]
	}

	if description == "" {
		description = fmt.Sprintf("Manual snapshot %s", opts.Type)
	}

	snap, err := mgr.Create(description, opts)
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "Snapshot created successfully.\nID: %s\nBackend: %s\n", snap.ID, snap.Backend)
	return nil
}

func init() {
	SnapshotCreateCmd.Flags().StringVarP(&snapshotCreateType, "type", "t", "single", "Snapshot type (single, pre, post, ondemand)")
	SnapshotCreateCmd.Flags().BoolVar(&snapshotCreateIncludeHome, "include-home", false, "Include home directory")
	SnapshotCreateCmd.Flags().StringSliceVar(&snapshotCreateTags, "tags", nil, "Tags for the snapshot")
}
