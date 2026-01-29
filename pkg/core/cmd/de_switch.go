package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/config"
	pkgConfig "github.com/theshedman/shedman/pkg/config"
	"github.com/theshedman/shedman/pkg/de"
	"github.com/theshedman/shedman/pkg/system"
	"github.com/theshedman/shedman/pkg/tui"
)

var (
	switchNoSnapshot bool
	switchKeepOld    bool
)

var deSwitchCmd = &cobra.Command{
	Use:   "switch <de-id>",
	Short: "Switch to a different desktop environment",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		deName := args[0]

		cfg, err := config.LoadDefault()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
			os.Exit(1)
		}

		engine, err := NewEngineWithConfig(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to initialize engine: %v\n", err)
			os.Exit(1)
		}

		snapMgr := engine.GetSnapshotManager()
		svcMgr := system.NewSystemdManager()

		mgr := de.New(engine)
		mgr.SetSnapshotManager(snapMgr)
		mgr.SetServiceManager(svcMgr)

		home, _ := os.UserHomeDir()
		statePath := fmt.Sprintf("%s/.local/state/shedman/configs.json", home)
		stateMgr := pkgConfig.NewJSONStateManager(statePath)
		_ = stateMgr.Load()

		configEng := pkgConfig.NewConfigEngine(
			stateMgr,
			pkgConfig.NewFileBackupManager(),
			pkgConfig.NewDiffer(),
			tui.NewConflictResolver(),
			pkgConfig.NewPacmanSourceProvider(nil),
		)
		applier := pkgConfig.NewDefaultApplier(configEng)
		mgr.SetConfigApplier(applier)

		yes, _ := cmd.Flags().GetBool("yes")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		opts := de.SwitchOptions{
			NoSnapshot: switchNoSnapshot,
			KeepOld:    switchKeepOld,
			NoConfirm:  yes,
			DryRun:     dryRun,
		}

		fmt.Printf("Switching Desktop Environment to %s...\n", deName)
		if err := mgr.Switch(cmd.Context(), deName, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error switching DE: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Successfully switched Desktop Environment.")
		if !switchKeepOld {
			fmt.Println("Note: You may need to log out or reboot for changes to take effect.")
		}
	},
}

func init() {
	deSwitchCmd.Flags().BoolVar(&switchNoSnapshot, "no-snapshot", false, "Skip pre-switch snapshot")
	deSwitchCmd.Flags().BoolVar(&switchKeepOld, "keep-old", false, "Do not remove current DE")
}
