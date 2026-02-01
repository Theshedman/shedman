package cmd

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/internal/output"
	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/snapshot"
)

var (
	updateShedOS      bool
	updateOfficial    bool
	updateAUR         bool
	updateYes         bool
	updateRefresh     bool
	updateDelta       bool
	updateLimitRate   string
	updateRetry       int
	updateTimeout     int
	updateIgnore      []string
	updateIgnoreGroup []string
)

var UpdateCmd = NewUpdateCmd()

func NewUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update [packages...]",
		Short: "Update system and installed packages",
		Long: `Update the system by synchronizing package databases and upgrading installed packages.
This command is equivalent to 'pacman -Syu' but handles all configured backends (ShedOS, Official, AUR).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadDefault()
			if err != nil {
				output.Warning("Failed to load config, using defaults: %v", err)
				cfg = config.Default()
			}

			eng, err := NewEngineWithConfig(cfg)
			if err != nil {
				return fmt.Errorf("failed to initialize engine: %w", err)
			}

			// Check global noconfirm flag (defined in root command)
			noconfirm, _ := cmd.Flags().GetBool("noconfirm")
			// Combine local yes flag with global noconfirm
			shouldNoConfirm := updateYes || noconfirm

			delta := updateDelta
			if !delta && cfg.Network.DeltaUpdates {
				delta = true
			}
			limitRate := updateLimitRate
			if limitRate == "" {
				limitRate = cfg.Network.LimitRate
			}
			retry := updateRetry
			if retry == 0 {
				retry = cfg.Network.Retry
			}
			timeout := updateTimeout
			if timeout == 0 {
				timeout = cfg.Network.Timeout
			}

			opts := core.UpgradeOptions{
				Refresh:      updateRefresh,
				NoConfirm:    shouldNoConfirm,
				IgnorePkgs:   mergeUnique(updateIgnore, cfg.Packages.IgnorePkg, cfg.Packages.HoldPkg),
				IgnoreGroups: mergeUnique(updateIgnoreGroup, cfg.Packages.IgnoreGroup),
				Delta:        delta,
				LimitRate:    limitRate,
				Retry:        retry,
				Timeout:      timeout,
			}

			if updateShedOS {
				opts.TargetBackends = append(opts.TargetBackends, "shedrepo")
			}
			if updateOfficial {
				opts.TargetBackends = append(opts.TargetBackends, "pacman", "libalpm")
			}
			if updateAUR {
				opts.TargetBackends = append(opts.TargetBackends, "aur")
			}

			return RunUpdate(eng, cmd.OutOrStdout(), args, opts)
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&updateShedOS, "shedos", false, "Update ShedOS repository packages only")
	flags.BoolVar(&updateOfficial, "official", false, "Update official packages only")
	flags.BoolVar(&updateAUR, "aur", false, "Update AUR packages only")
	flags.BoolVarP(&updateYes, "yes", "y", false, "Skip confirmation")
	flags.BoolVar(&updateRefresh, "refresh", false, "Force refresh sync databases before upgrade")
	flags.BoolVar(&updateDelta, "delta", false, "Enable delta updates when available")
	flags.StringVar(&updateLimitRate, "limit-rate", "", "Limit download rate (e.g. 500K, 2M)")
	flags.IntVar(&updateRetry, "retry", 0, "Number of download retries")
	flags.IntVar(&updateTimeout, "timeout", 0, "Network timeout in seconds")
	flags.StringSliceVar(&updateIgnore, "ignore", nil, "Ignore specific package upgrade (can be used multiple times)")
	flags.StringSliceVar(&updateIgnoreGroup, "ignoregroup", nil, "Ignore specific package group (can be used multiple times)")

	return cmd
}

func mergeUnique(lists ...[]string) []string {
	seen := make(map[string]bool)
	var merged []string
	for _, list := range lists {
		for _, item := range list {
			item = strings.TrimSpace(item)
			if item == "" || seen[item] {
				continue
			}
			seen[item] = true
			merged = append(merged, item)
		}
	}
	return merged
}

func RunUpdate(eng *core.Engine, w io.Writer, pkgs []string, opts core.UpgradeOptions) error {
	if err := maybeCreateAutoSnapshot(eng, w); err != nil {
		return err
	}

	if opts.Refresh {
		eng.SetSyncForceRefresh(true)
	}

	_, _ = fmt.Fprintln(w, "Synchronizing package databases...")

	if err := eng.Sync(); err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	_, _ = fmt.Fprintln(w, "Starting full system upgrade...")

	if err := eng.Upgrade(pkgs, opts); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	return nil
}

func maybeCreateAutoSnapshot(eng *core.Engine, w io.Writer) error {
	if eng == nil {
		return nil
	}
	cfg := eng.GetConfig()
	if cfg == nil || !cfg.Snapshot.AutoBeforeUpdate {
		return nil
	}

	mgr := eng.GetSnapshotManager()
	if mgr == nil {
		_, _ = fmt.Fprintln(w, "Warning: snapshot manager not available, skipping auto snapshot.")
		return nil
	}

	_, _ = fmt.Fprintln(w, "Creating pre-update snapshot...")
	_, err := mgr.Create(context.Background(), "pre-update", snapshot.CreateOptions{Type: "pre"})
	if err != nil {
		return fmt.Errorf("auto snapshot failed: %w", err)
	}
	return nil
}
