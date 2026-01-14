package cmd

import (
	"fmt"
	"io"
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
	syncForce    bool
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
		configFile, _ := cmd.Flags().GetString("config")
		cfg, err := config.Load(configFile)
		if err != nil {
			return err
		}

		fsCache := core.NewFileSystemCache()

		var backendList []core.PackageBackend
		syncAll := !syncOfficial && !syncAUR && !syncShedOS

		if syncAll || syncOfficial {
			pacmanBackend, err := pacman.New()
			if err != nil {
				if syncOfficial {
					return fmt.Errorf("pacman backend not available: %w", err)
				}
				output.Warning("Pacman backend not available: %v", err)
			} else {
				backendList = append(backendList, pacmanBackend)
			}
		}
		if syncAll || syncAUR {
			if !core.IsAURAvailable() {
				if syncAUR {
					return core.ErrAURNotAvailable
				}
				// Skip silently on non-Arch unless debug
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
			timeout := 30 * time.Second
			if cfg.Network.Timeout > 0 {
				timeout = time.Duration(cfg.Network.Timeout) * time.Second
			}
			if cfg.Mirrors.ShedOS != nil && len(cfg.Mirrors.ShedOS) > 0 {
				backendList = append(backendList, shedrepo.NewWithMirrors(cfg.Mirrors.ShedOS, fsCache, timeout))
			} else {
				backendList = append(backendList, shedrepo.New(fsCache, timeout))
			}
		}

		engine := core.NewEngine()
		verbose, _ := cmd.Flags().GetBool("verbose")
		for _, b := range backendList {
			engine.AddBackend(b)
		}

		quiet, _ := cmd.Flags().GetBool("quiet")
		if quiet {
			return RunSync(engine, io.Discard)
		}

		if verbose {
			for _, b := range backendList {
				cmd.Printf("  Syncing %s...\n", b.Name())
			}
		}

		if err := RunSync(engine, cmd.OutOrStdout()); err != nil {
			return err
		}

		if !quiet {
			output.Success("Sync complete.")
		}
		return nil
	},
}

// RunSync executes the sync logic
func RunSync(eng *core.Engine, w io.Writer) error {
	fmt.Fprintln(w, "Synchronizing package databases...")
	return eng.Sync()
}

func init() {
	SyncCmd.Flags().BoolVar(&syncOfficial, "official", false, "Sync official Arch repositories only")
	SyncCmd.Flags().BoolVar(&syncAUR, "aur", false, "Sync AUR cache only")
	SyncCmd.Flags().BoolVar(&syncShedOS, "shedos", false, "Sync ShedOS repository only")
	SyncCmd.Flags().BoolVar(&syncRefresh, "refresh", false, "Force refresh even if cache exists")
	SyncCmd.Flags().BoolVar(&syncForce, "force", false, "Force refresh (alias for --refresh)")
}
