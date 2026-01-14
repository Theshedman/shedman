package cmd

import (
	"os"
	"path/filepath"

	"github.com/theshedman/shedman/internal/config"
	pkgconfig "github.com/theshedman/shedman/pkg/config"
	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/core/providers/pacman"
	"github.com/theshedman/shedman/pkg/snapshot"
	"github.com/theshedman/shedman/pkg/tui"
)

// NewEngineWithConfig creates an Engine with backends auto-detected from config.
func NewEngineWithConfig(cfg *config.Config) (*core.Engine, error) {
	if cfg == nil {
		var err error
		cfg, err = config.LoadDefault()
		if err != nil {
			// If loading fails, fallback to defaults
			cfg = config.Default()
		}
	}

	e := core.NewEngine()

	officialBackend, err := DetectBackendWithConfig(&cfg.Backend)
	if err == nil && officialBackend != nil {
		e.SetOfficialBackend(officialBackend)
	}

	// Initialize Snapshot Manager
	snapFactory := snapshot.NewFactory(cfg)
	if snapMgr, err := snapFactory.GetManager(); err == nil {
		e.SetSnapshotManager(snapMgr)
	}

	return e, nil
}

// DetectBackendWithConfig detects backend based on config.
func DetectBackendWithConfig(cfg *config.BackendConfig) (core.OfficialBackend, error) {
	// Only pacman supported for now
	return CreatePacmanBackend(cfg)
}

// CreatePacmanBackend creates a pacman backend.
func CreatePacmanBackend(cfg *config.BackendConfig) (core.OfficialBackend, error) {
	c := pacman.DefaultConfig()
	if cfg != nil && cfg.BinaryPath != "" {
		c.BinaryPath = cfg.BinaryPath
	}
	return pacman.NewWithConfig(c)
}

// CreateAURInstaller creates an AUR installer with backend.
func CreateAURInstaller(cfg *config.Config) *core.AURInstaller {
	// Detect backend
	backend, _ := DetectBackendWithConfig(&cfg.Backend)
	return core.NewAURInstallerWithBackend(cfg, backend)
}

// CreateConfigEngine creates a ConfigEngine with default implementations
func CreateConfigEngine() *pkgconfig.ConfigEngine {
	home, _ := os.UserHomeDir() // Error ignored for factory defaults
	statePath := filepath.Join(home, ".local", "state", "shedman", "configs.json")

	stateMgr := pkgconfig.NewJSONStateManager(statePath)
	// Auto-load state
	_ = stateMgr.Load()

	backupMgr := pkgconfig.NewFileBackupManager()
	differ := pkgconfig.NewDiffer()
	resolver := tui.NewConflictResolver()
	provider := pkgconfig.NewPacmanSourceProvider(nil)

	return pkgconfig.NewConfigEngine(stateMgr, backupMgr, differ, resolver, provider)
}
