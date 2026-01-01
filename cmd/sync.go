package cmd

import (
	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/pkg/shedman"
	"github.com/theshedman/shedman/pkg/shedman/backends"
)

var (
	syncOfficial bool
	syncAUR      bool
	syncShedOS   bool
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync package databases",
	Long: `Synchronize package databases from configured sources.

By default, syncs all databases. Use flags to sync specific sources:
  --official    Sync official Arch repositories only
  --aur         Sync AUR cache only
  --shedos      Sync ShedOS repository only`,
	RunE: func(cmd *cobra.Command, args []string) error {
		engine := shedman.NewEngine()

		// Determine which backends to use
		syncAll := !syncOfficial && !syncAUR && !syncShedOS

		if syncAll || syncOfficial {
			engine.AddBackend(backends.NewPacmanBackend())
		}
		if syncAll || syncAUR {
			engine.AddBackend(backends.NewAURBackend())
		}
		if syncAll || syncShedOS {
			engine.AddBackend(backends.NewShedRepoBackend())
		}

		if !quietFlag {
			cmd.Println("Synchronizing package databases...")
		}

		if err := engine.Sync(); err != nil {
			return err
		}

		if !quietFlag {
			cmd.Println("Sync complete.")
		}

		if verboseFlag {
			cmd.Println("Verbose: All backends synced successfully.")
		}

		return nil
	},
}

func init() {
	syncCmd.Flags().BoolVar(&syncOfficial, "official", false, "Sync official Arch repositories only")
	syncCmd.Flags().BoolVar(&syncAUR, "aur", false, "Sync AUR cache only")
	syncCmd.Flags().BoolVar(&syncShedOS, "shedos", false, "Sync ShedOS repository only")
}
