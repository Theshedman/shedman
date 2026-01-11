// Package shedman provides the core engine for package management.
package core

import (
	"fmt"
	"strings"
	"sync"

	"github.com/theshedman/shedman/internal/config"
)

// Engine orchestrates package management operations across multiple backends.
type Engine struct {
	backends        []PackageBackend
	officialBackend OfficialBackend
	config          *config.Config
}

// NewEngine creates a new Engine with no backends configured.
func NewEngine() *Engine {
	return &Engine{
		backends: []PackageBackend{},
	}
}

// NewEngineWithBackend creates an Engine with a specific official backend.
// Useful for testing or when you want to provide a pre-configured backend.
func NewEngineWithBackend(b OfficialBackend) *Engine {
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
func (e *Engine) GetOfficialBackend() OfficialBackend {
	return e.officialBackend
}

// SetOfficialBackend sets the official backend.
// This is useful for late initialization or testing.
func (e *Engine) SetOfficialBackend(b OfficialBackend) {
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
func (e *Engine) Install(pkgs []string, opts InstallOptions) error {
	if e.officialBackend == nil {
		return ErrBackendNotFound
	}
	// Check if backend supports package management
	pm, ok := e.officialBackend.(PackageManager)
	if !ok {
		return fmt.Errorf("backend %s does not support package management", e.officialBackend.Name())
	}
	return pm.Install(pkgs, opts)
}

// InstallFile installs a local package file (wraps InstallLocal).
func (e *Engine) InstallFile(path string, opts InstallOptions) error {
	if e.officialBackend == nil {
		return ErrBackendNotFound
	}

	// Check if backend supports local package installation
	installer, ok := e.officialBackend.(LocalInstaller)
	if !ok {
		return fmt.Errorf("backend %s does not support local package installation", e.officialBackend.Name())
	}

	return installer.InstallLocal(path, opts)
}

// Remove removes packages using the official backend.
func (e *Engine) Remove(pkgs []string, opts RemoveOptions) error {
	if e.officialBackend == nil {
		return ErrBackendNotFound
	}
	// Check if backend supports package removal
	pm, ok := e.officialBackend.(PackageManager)
	if !ok {
		return fmt.Errorf("backend %s does not support package removal", e.officialBackend.Name())
	}
	return pm.Remove(pkgs, opts)
}

// Upgrade upgrades packages across all supported backends.
func (e *Engine) Upgrade(pkgs []string, opts UpgradeOptions) error {
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
			Upgrade(pkgs []string, opts UpgradeOptions) error
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
			return ErrBackendNotFound
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
	// Check if backend supports package checking
	pm, ok := e.officialBackend.(PackageManager)
	if !ok {
		return false
	}
	return pm.IsInstalled(name)
}

// Info returns detailed information about a package from available backends.
func (e *Engine) Info(pkgName string) (*PackageInfo, error) {
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
			Info(pkgName string) (*PackageInfo, error)
		}

		if informer, ok := b.(Informer); ok {
			if info, err := informer.Info(pkgName); err == nil {
				return info, nil
			}
		}
	}

	return nil, ErrPackageNotFound
}

// Search searches for packages across all available backends.
func (e *Engine) Search(query string) ([]PackageInfo, error) {
	var allResults []PackageInfo
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, b := range e.backends {
		wg.Add(1)
		go func(backend PackageBackend) {
			defer wg.Done()

			// Check if backend supports searching
			searchable, ok := backend.(Searchable)
			if !ok {
				return
			}

			results, err := searchable.Search(query)
			if err != nil {
				// Log error but continue
				return
			}

			mu.Lock()
			// Append source to backend if not already set (impl specific, but good practice here if needed)
			allResults = append(allResults, results...)
			mu.Unlock()
		}(b)
	}

	wg.Wait()
	return allResults, nil
}
