// Package shedman provides the core engine for package management.
package shedman

import (
	"fmt"
	"strings"
	"sync"

	"github.com/theshedman/shedman/pkg/shedman/backend"
	"github.com/theshedman/shedman/pkg/shedman/backend/pacman"
	"github.com/theshedman/shedman/pkg/shedman/config"
	"github.com/theshedman/shedman/pkg/shedman/pkgdb"
)

// Engine orchestrates package management operations across multiple backends.
type Engine struct {
	backends        []PackageBackend
	officialBackend backend.OfficialBackend
	config          *config.Config
}

// NewEngine creates a new Engine with no backends configured.
func NewEngine() *Engine {
	return &Engine{
		backends: []PackageBackend{},
	}
}

// NewEngineWithConfig creates an Engine with backends auto-detected from config.
// This is the preferred way to create an Engine for production use.
func NewEngineWithConfig(cfg *config.Config) (*Engine, error) {
	e := &Engine{
		backends: []PackageBackend{},
		config:   cfg,
	}

	// Detect and initialize the official backend
	officialBackend, err := backend.DetectBackendWithConfig(&cfg.Backend)
	if err == nil && officialBackend != nil {
		e.officialBackend = officialBackend
		// Also add to backends list for Sync compatibility
		e.backends = append(e.backends, officialBackend)
	}

	return e, nil
}

// NewEngineWithBackend creates an Engine with a specific official backend.
// Useful for testing or when you want to provide a pre-configured backend.
func NewEngineWithBackend(b backend.OfficialBackend) *Engine {
	e := &Engine{
		backends:        []PackageBackend{},
		officialBackend: b,
	}
	if b != nil {
		e.backends = append(e.backends, b)
	}
	return e
}

// AddBackend adds a package backend to the engine.
func (e *Engine) AddBackend(b PackageBackend) {
	e.backends = append(e.backends, b)
}

// GetOfficialBackend returns the detected official package manager backend.
// Returns nil if no official backend is available (e.g., on unsupported systems).
func (e *Engine) GetOfficialBackend() backend.OfficialBackend {
	return e.officialBackend
}

// SetOfficialBackend sets the official backend.
// This is useful for late initialization or testing.
func (e *Engine) SetOfficialBackend(b backend.OfficialBackend) {
	e.officialBackend = b
	// Ensure it's in the backends list
	for _, existing := range e.backends {
		if existing == b {
			return
		}
	}
	e.backends = append(e.backends, b)
}

// GetConfig returns the engine's configuration.
func (e *Engine) GetConfig() *config.Config {
	return e.config
}

// IsOfficialBackendAvailable returns true if an official backend is configured.
func (e *Engine) IsOfficialBackendAvailable() bool {
	return e.officialBackend != nil && e.officialBackend.IsAvailable()
}

// Sync synchronizes all configured backends.
// Sync synchronizes all configured backends in parallel.
func (e *Engine) Sync() error {
	var errors []string
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, b := range e.backends {
		wg.Add(1)
		go func(backend PackageBackend) {
			defer wg.Done()
			if err := backend.Sync(); err != nil {
				mu.Lock()
				errors = append(errors, fmt.Sprintf("%s: %v", backend.Name(), err))
				mu.Unlock()
			}
		}(b)
	}

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("sync failed for backends: %s", strings.Join(errors, "; "))
	}

	return nil
}

// Install installs packages using the official backend.
// Returns an error if no official backend is available.
func (e *Engine) Install(pkgs []string, opts backend.InstallOptions) error {
	if e.officialBackend == nil {
		return backend.ErrBackendNotFound
	}
	return e.officialBackend.Install(pkgs, opts)
}

// InstallFile installs a local package file (wraps InstallLocal).
func (e *Engine) InstallFile(path string, opts backend.InstallOptions) error {
	if e.officialBackend == nil {
		return backend.ErrBackendNotFound
	}
	return e.officialBackend.InstallLocal(path, opts)
}

// Remove removes packages using the official backend.
func (e *Engine) Remove(pkgs []string, opts backend.RemoveOptions) error {
	if e.officialBackend == nil {
		return backend.ErrBackendNotFound
	}
	return e.officialBackend.Remove(pkgs, opts)
}

// Upgrade upgrades packages across all supported backends.
func (e *Engine) Upgrade(pkgs []string, opts backend.UpgradeOptions) error {
	var errors []string

	// Track whether we found any backend capable of upgrading
	upgradableFound := false

	// Iterate all backends
	for _, b := range e.backends {
		// Check if backend is targeted
		if len(opts.TargetBackends) > 0 {
			targeted := false
			for _, target := range opts.TargetBackends {
				if strings.EqualFold(b.Name(), target) {
					targeted = true
					break
				}
			}
			if !targeted {
				continue
			}
		}

		// Check if backend supports Upgrading via interface assertion
		type Upgrader interface {
			Upgrade(pkgs []string, opts backend.UpgradeOptions) error
		}

		if upgrader, ok := b.(Upgrader); ok {
			upgradableFound = true
			if err := upgrader.Upgrade(pkgs, opts); err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", b.Name(), err))
			}
		}
	}

	if !upgradableFound {
		if e.officialBackend == nil {
			return backend.ErrBackendNotFound
		}
		// Fallback to official backend if no generic Upgrader found (though official usually implements it)
		return e.officialBackend.Upgrade(pkgs, opts)
	}

	if len(errors) > 0 {
		return fmt.Errorf("upgrade failed for backends: %s", strings.Join(errors, "; "))
	}

	return nil
}

// IsInstalled checks if a package is installed via the official backend.
func (e *Engine) IsInstalled(name string) bool {
	if e.officialBackend == nil {
		return false
	}
	return e.officialBackend.IsInstalled(name)
}

// Info returns detailed information about a package from available backends.
func (e *Engine) Info(pkgName string) (*pkgdb.PackageInfo, error) {
	// 1. Try official backend first (local + sync)
	if e.officialBackend != nil {
		if info, err := e.officialBackend.Info(pkgName); err == nil {
			return info, nil
		}
	}

	// 2. Iterate other backends (e.g. AUR, ShedRepo)
	for _, b := range e.backends {
		if b == e.officialBackend {
			continue // Already checked
		}

		type Informer interface {
			Info(pkgName string) (*pkgdb.PackageInfo, error)
		}

		if informer, ok := b.(Informer); ok {
			if info, err := informer.Info(pkgName); err == nil {
				return info, nil
			}
		}
	}

	return nil, backend.ErrPackageNotFound
}

// DetectBackend auto-detects and returns the appropriate official backend.
// This is a convenience method that wraps backend.DetectBackendWithConfig.
func DetectBackend(cfg *config.Config) (backend.OfficialBackend, error) {
	if cfg == nil {
		cfg = config.Default()
	}
	return backend.DetectBackendWithConfig(&cfg.Backend)
}

// CreatePacmanBackend creates a pacman backend with optional config.
// Returns an error if pacman is not available.
func CreatePacmanBackend(cfg *config.BackendConfig) (backend.OfficialBackend, error) {
	c := pacman.DefaultConfig()
	if cfg != nil && cfg.BinaryPath != "" {
		c.BinaryPath = cfg.BinaryPath
	}
	return pacman.NewWithConfig(c)
}
