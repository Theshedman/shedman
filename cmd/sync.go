package cmd

import (
"github.com/spf13/cobra"
"github.com/theshedman/shedman/pkg/shedman"
"github.com/theshedman/shedman/pkg/shedman/backends"
"github.com/theshedman/shedman/pkg/shedman/cache"
"github.com/theshedman/shedman/pkg/shedman/config"
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
			backendList = append(backendList, backends.NewPacmanBackend())
		}
		if syncAll || syncAUR {
			backendList = append(backendList, backends.NewAURBackend(c))
		}
		if syncAll || syncShedOS {
			// Use mirrors from config
			shedRepo := backends.NewShedRepoBackendWithMirrors(cfg.ShedRepoMirrors, c)
			if syncRefresh {
				shedRepo.SetForceRefresh(true)
			}
			backendList = append(backendList, shedRepo)
		}

		// Debug output
		if debugFlag {
			cmd.Printf("[DEBUG] Config file: %s\n", configFile)
			cmd.Printf("[DEBUG] Cache directory: %s\n", c.GetDir())
			cmd.Printf("[DEBUG] ShedRepo mirrors: %v\n", cfg.ShedRepoMirrors)
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
			cmd.Println("Synchronizing package databases...")
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
			cmd.Println("Sync complete.")
		}

		return nil
	},
}

func init() {
	syncCmd.Flags().BoolVar(&syncOfficial, "official", false, "Sync official Arch repositories only")
	syncCmd.Flags().BoolVar(&syncAUR, "aur", false, "Sync AUR cache only")
	syncCmd.Flags().BoolVar(&syncShedOS, "shedos", false, "Sync ShedOS repository only")
	syncCmd.Flags().BoolVar(&syncRefresh, "refresh", false, "Force refresh even if cache exists")
}
