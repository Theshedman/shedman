package cmd

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/snapshot"
)

var (
	cleanAll       bool
	cleanCache     bool
	cleanOrphans   bool
	cleanSnapshots bool
	cleanKeep      int
)

// CleanCmd represents the clean command
var CleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean the package cache",
	Long:  `Remove uninstalled or all packages from the cache to free up disk space. Use --keep to retain recent versions.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefault()
		if err != nil {
			cfg = config.Default()
		}

		eng, err := NewEngineWithConfig(cfg)
		if err != nil {
			return fmt.Errorf("failed to initialize engine: %w", err)
		}

		opts := CleanOptions{
			All:       cleanAll,
			Cache:     cleanCache,
			Orphans:   cleanOrphans,
			Snapshots: cleanSnapshots,
			Keep:      cleanKeep,
		}

		if err := RunClean(cmd.Context(), eng, cmd.OutOrStdout(), cfg, opts); err != nil {
			// Return error for RunE after logging
			return fmt.Errorf("clean failed: %w", err)
		}

		fmt.Println("Clean completed successfully.")
		// Print success message to writer
		return nil
	},
}

func init() {
	CleanCmd.Flags().BoolVar(&cleanAll, "all", false, "Clean cache, orphans, and snapshots")
	CleanCmd.Flags().BoolVar(&cleanCache, "cache", false, "Clean package cache only")
	CleanCmd.Flags().BoolVar(&cleanOrphans, "orphans", false, "Remove orphaned packages")
	CleanCmd.Flags().BoolVar(&cleanSnapshots, "snapshots", false, "Prune old snapshots")
	CleanCmd.Flags().IntVar(&cleanKeep, "keep", 0, "Number of recent versions to keep in cache")
}

// CleanOptions holds clean command options.
type CleanOptions struct {
	All       bool
	Cache     bool
	Orphans   bool
	Snapshots bool
	Keep      int
}

func RunClean(ctx context.Context, eng *core.Engine, w io.Writer, cfg *config.Config, opts CleanOptions) error {
	if opts.All {
		opts.Cache = true
		opts.Orphans = true
		opts.Snapshots = true
	}

	if !opts.Cache && !opts.Orphans && !opts.Snapshots {
		opts.Cache = true
	}

	if opts.Cache {
		_, _ = fmt.Fprintln(w, "Cleaning package cache...")

		keep := opts.Keep
		if keep == 0 && cfg != nil && cfg.Cache.CleanKeep > 0 {
			keep = cfg.Cache.CleanKeep
		}

		cleanOpts := core.CleanOptions{
			All:  opts.All,
			Keep: keep,
		}
		if err := eng.CleanCache(cleanOpts); err != nil {
			return err
		}
	}

	if opts.Orphans {
		_, _ = fmt.Fprintln(w, "Removing orphan packages...")
		orphans, err := eng.ListOrphans()
		if err != nil {
			return err
		}
		if len(orphans) == 0 {
			_, _ = fmt.Fprintln(w, "No orphans found.")
		} else if err := eng.RemoveOrphans(orphans); err != nil {
			return err
		}
	}

	if opts.Snapshots {
		_, _ = fmt.Fprintln(w, "Pruning snapshots...")
		mgr := eng.GetSnapshotManager()
		if mgr == nil {
			_, _ = fmt.Fprintln(w, "Snapshot manager not available.")
		} else {
			prune := snapshot.PruneOptions{}
			if cfg != nil {
				prune.KeepLast = cfg.Snapshot.KeepLocal
				prune.KeepScheduled = cfg.Snapshot.KeepScheduled
			}
			if err := mgr.Prune(ctx, prune); err != nil {
				return err
			}
		}
	}

	return nil
}
