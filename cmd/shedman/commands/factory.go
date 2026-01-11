package commands

import (
	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/core/providers/pacman"
)

// NewEngineWithConfig creates an Engine with backends auto-detected from config.
func NewEngineWithConfig(cfg *config.Config) (*core.Engine, error) {
	e := core.NewEngine()
	// Need to set config manually or add SetConfig to engine?
	// Engine struct has config field but NewEngine doesn't take it.
	// Since we are moving logic, we might need a setter or modified constructor in core.
	// For now, let's assume we can modify it or access it if exported.
	// Wait, Engine struct fields are private (lowercase backends/config).
	// But NewEngine returns *Engine.
	// We need a way to set config.
	// Let's check Engine API.

	// Assuming core.DetectBackendWithConfig existed? No, it was removed.
	// We implement detection here.

	officialBackend, err := DetectBackendWithConfig(&cfg.Backend)
	if err == nil && officialBackend != nil {
		e.SetOfficialBackend(officialBackend)
	}

	// We cannot set e.config directly if private.
	// core/engine.go has GetConfig(). Does it have SetConfig?
	// If not, we should adding it or passing it to NewEngine.
	// Checking previous engine.go content: NewEngine() didn't take config.
	// NewEngineWithConfig() did set it.
	// So we need to add SetConfig to Engine in a separate step or modify NewEngine signature.
	// Let's add SetConfig to Engine first.

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

// CreateShedOSInstaller creates a ShedOS installer with backend.
func CreateShedOSInstaller(cfg *config.Config) *core.ShedOSInstaller {
	// Detect backend
	backend, _ := DetectBackendWithConfig(&cfg.Backend)
	return core.NewShedOSInstallerWithBackend(cfg, backend)
}
