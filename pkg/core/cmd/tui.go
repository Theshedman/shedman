package cmd

import (
	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/de"
	"github.com/theshedman/shedman/pkg/snapshot"
	"github.com/theshedman/shedman/pkg/tui"
)

// TUICmd is the tui command
var TUICmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch interactive TUI dashboard",
	Long:  "Launch an interactive terminal user interface to manage packages, snapshots, desktop environments, and system health.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load Config
		cfg, err := config.LoadDefault()
		if err != nil {
			cfg = config.Default()
		}

		// Initialize Engine
		backend, err := DetectBackendWithConfig(&cfg.Backend)
		if err != nil {
			return err
		}
		eng := core.NewEngineWithBackend(backend)
		eng.SetConfig(cfg)

		// Initialize Snapshot Manager
		factory := snapshot.NewFactory(cfg)
		sm, err := factory.GetManager()
		if err == nil {
			eng.SetSnapshotManager(sm)
		}

		// Initialize DE Manager
		deMgr := de.New(eng)
		if sm != nil {
			deMgr.SetSnapshotManager(sm)
		}

		app := tui.New(eng, deMgr)
		return app.Run()
	},
}
