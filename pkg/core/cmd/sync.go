package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/internal/output"
	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/core/providers/aur"
	"github.com/theshedman/shedman/pkg/core/providers/pacman"
	shedrepo "github.com/theshedman/shedman/pkg/core/providers/shed"
)

var (
	syncOfficial bool
	syncAUR      bool
	syncShedOS   bool
	syncRefresh  bool
)

var SyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync package databases",
	Long: `Synchronize package databases from configured sources.

By default, syncs all databases. Use flags to sync specific sources:
  --official    Sync official Arch repositories only
  --aur         Sync AUR cache only
  --shedos      Sync ShedOS repository only
  --refresh     Force refresh even if cache exists`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration
		configFile, _ := cmd.Flags().GetString("config")
		cfg, err := config.Load(configFile)
		if err != nil {
			return err
		}

		fsCache := core.NewFileSystemCache()

		// Collect backends based on flags
		var backendList []core.PackageBackend
		syncAll := !syncOfficial && !syncAUR && !syncShedOS

		if syncAll || syncOfficial {
			pacmanBackend, err := pacman.New()
			if err != nil {
				if syncOfficial {
					// Explicit --official flag, return error
					return fmt.Errorf("pacman backend not available: %w", err)
				}
				// syncAll - warn and continue
				output.Warning("Pacman backend not available: %v", err)
			} else {
				backendList = append(backendList, pacmanBackend)
			}
		}
		if syncAll || syncAUR {
			if !core.IsAURAvailable() {
				if syncAUR {
					// Explicit --aur flag, return error
					return core.ErrAURNotAvailable
				}
				// syncAll - just skip AUR silently on non-Arch systems
				debug, _ := cmd.Flags().GetBool("debug")
				if debug {
					output.Warning("Skipping AUR sync: not on Arch-based system")
				}
			} else {
				// Need pkgCache for AUR
				pkgCache := core.NewPackageFileCacheWithBackend(24*time.Hour, nil)
				backendList = append(backendList, aur.New(pkgCache))
			}
		}
		if syncAll || syncShedOS {
			// Use mirrors from config
			timeout := 30 * time.Second
			if cfg.Network.Timeout > 0 {
				timeout = time.Duration(cfg.Network.Timeout) * time.Second
			}
			// ShedOS backend
			if cfg.Mirrors.ShedOS != nil && len(cfg.Mirrors.ShedOS) > 0 {
				backendList = append(backendList, shedrepo.NewWithMirrors(cfg.Mirrors.ShedOS, fsCache, timeout))
			} else {
				backendList = append(backendList, shedrepo.New(fsCache, timeout))
			}
		}

		// Debug output
		debug, _ := cmd.Flags().GetBool("debug")
		if debug {
			configFile, _ := cmd.Flags().GetString("config")
			cmd.Printf("[DEBUG] Config file: %s\n", configFile)
			cmd.Printf("[DEBUG] Cache directory: %s\n", fsCache.GetDir())
			cmd.Printf("[DEBUG] ShedRepo mirrors: %v\n", cfg.Mirrors.ShedOS)
			cmd.Printf("[DEBUG] Backends to sync: %d\n", len(backendList))
			cmd.Printf("[DEBUG] Refresh mode: %v\n", syncRefresh)
			for _, b := range backendList {
				cmd.Printf("[DEBUG]   - %s\n", b.Name())
			}
		}

		// Dry-run mode
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		if dryRun {
			cmd.Println("Dry-run mode: would sync the following backends:")
			for _, b := range backendList {
				cmd.Printf("  - %s\n", b.Name())
			}
			return nil
		}

		quiet, _ := cmd.Flags().GetBool("quiet")
		if !quiet {
			output.Info("Synchronizing package databases to %s...", fsCache.GetDir())
		}

		// Sync each backend with verbose progress
		engine := core.NewEngine()
		verbose, _ := cmd.Flags().GetBool("verbose")
		for _, backend := range backendList {
			if verbose {
				cmd.Printf("  Syncing %s...\n", backend.Name())
			}
			engine.AddBackend(backend)
		}

		if err := engine.Sync(); err != nil {
			return err
		}

		if !quiet {
			output.Success("Sync complete.")
		}

		return nil
	},
}

func init() {
	SyncCmd.Flags().BoolVar(&syncOfficial, "official", false, "Sync official Arch repositories only")
	SyncCmd.Flags().BoolVar(&syncAUR, "aur", false, "Sync AUR cache only")
	SyncCmd.Flags().BoolVar(&syncShedOS, "shedos", false, "Sync ShedOS repository only")
	SyncCmd.Flags().BoolVar(&syncRefresh, "refresh", false, "Force refresh even if cache exists")
	SyncCmd.Flags().BoolVar(&syncRefresh, "force", false, "Force refresh (alias for --refresh)")
}
