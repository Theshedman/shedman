package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/internal/output"
	"github.com/theshedman/shedman/pkg/core"
)

var (
	updateShedOS      bool
	updateOfficial    bool
	updateAUR         bool
	updateYes         bool
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

			opts := core.UpgradeOptions{
				Refresh:      false, // We sync manually
				NoConfirm:    updateYes,
				IgnorePkgs:   updateIgnore,
				IgnoreGroups: updateIgnoreGroup,
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
	flags.StringSliceVar(&updateIgnore, "ignore", nil, "Ignore specific package upgrade (can be used multiple times)")
	flags.StringSliceVar(&updateIgnoreGroup, "ignoregroup", nil, "Ignore specific package group (can be used multiple times)")

	return cmd
}

// RunUpdate executes the update logic
func RunUpdate(eng *core.Engine, w io.Writer, pkgs []string, opts core.UpgradeOptions) error {
	fmt.Fprintln(w, "Synchronizing package databases...")

	if err := eng.Sync(); err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	fmt.Fprintln(w, "Starting full system upgrade...")

	if err := eng.Upgrade(pkgs, opts); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	return nil
}
