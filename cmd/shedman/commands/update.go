package commands

import (
	"fmt"

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
		Use:   "update",
		Short: "Update system and installed packages",
		Long: `Update the system by synchronizing package databases and upgrading installed packages.
This command is equivalent to 'pacman -Syu' but handles all configured backends (ShedOS, Official, AUR).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(args)
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

func runUpdate(args []string) error {
	// Load configuration
	cfg, err := config.LoadDefault()
	if err != nil {
		output.Warning("Failed to load config, using defaults: %v", err)
		cfg = config.Default()
	}

	// Determine specific package update vs full system
	// If args provided, we update specific packages.
	// If not, full system update.
	pkgs := args

	// Setup options
	opts := core.UpgradeOptions{
		// We sync manually via engine for parallelism, so we disable refresh in Upgrade
		Refresh:      false,
		NoConfirm:    updateYes,
		IgnorePkgs:   updateIgnore,
		IgnoreGroups: updateIgnoreGroup,
	}

	// Handle target backends
	if updateShedOS {
		opts.TargetBackends = append(opts.TargetBackends, "shedrepo")
	}
	if updateOfficial {
		opts.TargetBackends = append(opts.TargetBackends, "pacman", "libalpm") // Covers both implementations
	}
	if updateAUR {
		opts.TargetBackends = append(opts.TargetBackends, "aur")
	}

	// Initialize Engine to manage backends
	eng, err := NewEngineWithConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize engine: %w", err)
	}

	// 1. Sync databases first (Parallel)
	output.Info("Synchronizing package databases...")
	if err := eng.Sync(); err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	// 2. Perform Upgrade (Sequential/Atomic)
	output.Info("Starting full system upgrade...")

	if err := eng.Upgrade(pkgs, opts); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	return nil
}

func init() {
}
