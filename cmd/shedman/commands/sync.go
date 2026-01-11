package commands

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/cache"
	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/internal/output"
	"github.com/theshedman/shedman/pkg/backend"
	"github.com/theshedman/shedman/pkg/backend/aur"
	"github.com/theshedman/shedman/pkg/backend/pacman"
	"github.com/theshedman/shedman/pkg/backend/shedrepo"
	shedman "github.com/theshedman/shedman/pkg/core"
)

var (
	syncOfficial bool
	syncAUR      bool
	syncShedOS   bool
	syncRefresh  bool
)

var syncCmd = &cobra.Command{
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
		cfg, err := config.Load(configFile)
		if err != nil {
			return err
		}

		c := cache.NewFileSystemCache()

		// Collect backends based on flags
		var backendList []shedman.PackageBackend
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
			if !backend.IsAURAvailable() {
				if syncAUR {
					// Explicit --aur flag, return error
					return backend.ErrAURNotAvailable
				}
				// syncAll - just skip AUR silently on non-Arch systems
				if debugFlag {
					output.Warning("Skipping AUR sync: not on Arch-based system")
				}
			} else {
				backendList = append(backendList, aur.New(c))
			}
		}
		if syncAll || syncShedOS {
			// Use mirrors from config
			timeout := 30 * time.Second
			if cfg.Network.Timeout > 0 {
				timeout = time.Duration(cfg.Network.Timeout) * time.Second
			}
			shedRepo := shedrepo.NewWithMirrors(cfg.Mirrors.ShedOS, c, timeout)
			if syncRefresh {
				shedRepo.SetForceRefresh(true)
			}
			backendList = append(backendList, shedRepo)
		}

		// Debug output
		if debugFlag {
			cmd.Printf("[DEBUG] Config file: %s\n", configFile)
			cmd.Printf("[DEBUG] Cache directory: %s\n", c.GetDir())
			cmd.Printf("[DEBUG] ShedRepo mirrors: %v\n", cfg.Mirrors.ShedOS)
			cmd.Printf("[DEBUG] Backends to sync: %d\n", len(backendList))
			cmd.Printf("[DEBUG] Refresh mode: %v\n", syncRefresh)
			for _, b := range backendList {
				cmd.Printf("[DEBUG]   - %s\n", b.Name())
			}
		}

		// Dry-run mode
		if dryRunFlag {
			cmd.Println("Dry-run mode: would sync the following backends:")
			for _, b := range backendList {
				cmd.Printf("  - %s\n", b.Name())
			}
			return nil
		}

		if !quietFlag {
			output.Info("Synchronizing package databases...")
		}

		// Sync each backend with verbose progress
		engine := shedman.NewEngine()
		for _, backend := range backendList {
			if verboseFlag {
				cmd.Printf("  Syncing %s...\n", backend.Name())
			}
			engine.AddBackend(backend)
		}

		if err := engine.Sync(); err != nil {
			return err
		}

		if !quietFlag {
			output.Success("Sync complete.")
		}

		return nil
	},
}

func init() {
	syncCmd.Flags().BoolVar(&syncOfficial, "official", false, "Sync official Arch repositories only")
	syncCmd.Flags().BoolVar(&syncAUR, "aur", false, "Sync AUR cache only")
	syncCmd.Flags().BoolVar(&syncShedOS, "shedos", false, "Sync ShedOS repository only")
	syncCmd.Flags().BoolVar(&syncRefresh, "refresh", false, "Force refresh even if cache exists")
	syncCmd.Flags().BoolVar(&syncRefresh, "force", false, "Force refresh (alias for --refresh)")

	rootCmd.AddCommand(syncCmd)
}
