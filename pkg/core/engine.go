// Package shedman provides the core engine for package management.
package core

import (
	"fmt"
	"strings"
	"sync"

	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/pkg/snapshot"
)

// Engine orchestrates package management operations across multiple backends.
type Engine struct {
	backends         []PackageBackend
	officialBackend  OfficialBackend
	config           *config.Config
	snapshotManager  snapshot.Manager
	scheduler        snapshot.Scheduler
	keyManager       snapshot.KeyManager
	syncForceRefresh bool
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

// SetSnapshotManager sets the snapshot manager.
func (e *Engine) SetSnapshotManager(sm snapshot.Manager) {
	e.snapshotManager = sm
}

// GetSnapshotManager returns the snapshot manager.
func (e *Engine) GetSnapshotManager() snapshot.Manager {
	return e.snapshotManager
}

// SetScheduler sets the snapshot scheduler.
func (e *Engine) SetScheduler(s snapshot.Scheduler) {
	e.scheduler = s
}

// SetSyncForceRefresh toggles forced refresh for the next Sync call.
func (e *Engine) SetSyncForceRefresh(force bool) {
	e.syncForceRefresh = force
}

// GetScheduler returns the snapshot scheduler.
func (e *Engine) GetScheduler() snapshot.Scheduler {
	return e.scheduler
}

// SetKeyManager sets the snapshot key manager.
func (e *Engine) SetKeyManager(km snapshot.KeyManager) {
	e.keyManager = km
}

// GetKeyManager returns the snapshot key manager.
func (e *Engine) GetKeyManager() snapshot.KeyManager {
	return e.keyManager
}

// SetConfig sets the engine's configuration.
func (e *Engine) SetConfig(cfg *config.Config) {
	e.config = cfg
}

// GetConfig returns the engine's configuration.
func (e *Engine) GetConfig() *config.Config {
	return e.config
}

// IsOfficialBackendAvailable returns true if an official backend is configured.
func (e *Engine) IsOfficialBackendAvailable() bool {
	return e.officialBackend != nil && e.officialBackend.IsAvailable()
}

// Sync synchronizes all configured backends in parallel.
func (e *Engine) Sync() error {
	if e.syncForceRefresh {
		e.applySyncForceRefresh()
	}
	preSync := ""
	postSync := ""
	if e.config != nil {
		preSync = e.config.Hooks.PreSync
		postSync = e.config.Hooks.PostSync
	}

	if err := e.runHook(preSync, "pre-sync", nil); err != nil {
		return err
	}

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
		e.syncForceRefresh = false
		return fmt.Errorf("sync failed for backends: %s", strings.Join(errors, "; "))
	}

	if err := e.runHook(postSync, "post-sync", nil); err != nil {
		e.syncForceRefresh = false
		return err
	}

	e.syncForceRefresh = false
	return nil
}

func (e *Engine) applySyncForceRefresh() {
	for _, backend := range e.backends {
		if setter, ok := backend.(interface{ SetForceRefresh(bool) }); ok {
			setter.SetForceRefresh(true)
		}
	}
}

// Install installs packages using the official backend.
// Returns an error if no official backend is available.
func (e *Engine) Install(pkgs []string, opts InstallOptions) error {
	if e.officialBackend == nil {
		return ErrBackendNotFound
	}
	pm, ok := e.officialBackend.(PackageManager)
	if !ok {
		return fmt.Errorf("backend %s does not support package management", e.officialBackend.Name())
	}
	preInstall := ""
	postInstall := ""
	if e.config != nil {
		preInstall = e.config.Hooks.PreInstall
		postInstall = e.config.Hooks.PostInstall
	}

	if err := e.runHook(preInstall, "pre-install", pkgs); err != nil {
		return err
	}
	if err := pm.Install(pkgs, opts); err != nil {
		e.logTransaction("installed", pkgs, false)
		return err
	}
	if err := e.runHook(postInstall, "post-install", pkgs); err != nil {
		return err
	}
	e.logTransaction("installed", pkgs, true)
	return nil
}

// InstallFile installs a local package file (wraps InstallLocal).
func (e *Engine) InstallFile(path string, opts InstallOptions) error {
	if e.officialBackend == nil {
		return ErrBackendNotFound
	}

	installer, ok := e.officialBackend.(LocalInstaller)
	if !ok {
		return fmt.Errorf("backend %s does not support local package installation", e.officialBackend.Name())
	}

	preInstall := ""
	postInstall := ""
	if e.config != nil {
		preInstall = e.config.Hooks.PreInstall
		postInstall = e.config.Hooks.PostInstall
	}

	if err := e.runHook(preInstall, "pre-install", []string{path}); err != nil {
		return err
	}
	if err := installer.InstallLocal(path, opts); err != nil {
		e.logTransaction("installed", []string{path}, false)
		return err
	}
	if err := e.runHook(postInstall, "post-install", []string{path}); err != nil {
		return err
	}
	e.logTransaction("installed", []string{path}, true)
	return nil
}

// Remove removes packages using the official backend.
func (e *Engine) Remove(pkgs []string, opts RemoveOptions) error {
	if e.officialBackend == nil {
		return ErrBackendNotFound
	}
	pm, ok := e.officialBackend.(PackageManager)
	if !ok {
		return fmt.Errorf("backend %s does not support package removal", e.officialBackend.Name())
	}
	preRemove := ""
	postRemove := ""
	if e.config != nil {
		preRemove = e.config.Hooks.PreRemove
		postRemove = e.config.Hooks.PostRemove
	}

	if err := e.runHook(preRemove, "pre-remove", pkgs); err != nil {
		return err
	}
	if err := pm.Remove(pkgs, opts); err != nil {
		e.logTransaction("removed", pkgs, false)
		return err
	}
	if err := e.runHook(postRemove, "post-remove", pkgs); err != nil {
		return err
	}
	e.logTransaction("removed", pkgs, true)
	return nil
}

// Upgrade upgrades packages across all supported backends.
func (e *Engine) Upgrade(pkgs []string, opts UpgradeOptions) error {
	var errors []string

	// Track whether we found any backend capable of upgrading
	upgradableFound := false

	preUpgrade := ""
	postUpgrade := ""
	if e.config != nil {
		preUpgrade = e.config.Hooks.PreUpgrade
		postUpgrade = e.config.Hooks.PostUpgrade
	}

	if err := e.runHook(preUpgrade, "pre-upgrade", pkgs); err != nil {
		return err
	}

	// Iterate all backends
	for _, b := range e.backends {
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

		// Define Upgrader interface locally for type assertion
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
		// Fallback to official backend
		if err := e.officialBackend.Upgrade(pkgs, opts); err != nil {
			e.logTransaction("upgraded", pkgs, false)
			return err
		}
		if err := e.runHook(postUpgrade, "post-upgrade", pkgs); err != nil {
			return err
		}
		e.logTransaction("upgraded", pkgs, true)
		return nil
	}

	if len(errors) > 0 {
		e.logTransaction("upgraded", pkgs, false)
		return fmt.Errorf("upgrade failed for backends: %s", strings.Join(errors, "; "))
	}

	if err := e.runHook(postUpgrade, "post-upgrade", pkgs); err != nil {
		return err
	}

	e.logTransaction("upgraded", pkgs, true)
	return nil
}

// IsInstalled checks if a package is installed via the official backend.
func (e *Engine) IsInstalled(name string) bool {
	if e.officialBackend == nil {
		return false
	}
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

			searchable, ok := backend.(Searchable)
			if !ok {
				return
			}

			results, err := searchable.Search(query)
			if err != nil {
				// Continue on error
				return
			}

			mu.Lock()
			// Append source if missing
			allResults = append(allResults, results...)
			mu.Unlock()
		}(b)
	}

	wg.Wait()
	return allResults, nil
}

// CleanCache cleans the package cache.
func (e *Engine) CleanCache(opts CleanOptions) error {
	if e.officialBackend == nil {
		return ErrBackendNotFound
	}
	if m, ok := e.officialBackend.(Maintainer); ok {
		return m.CleanCache(opts)
	}
	return fmt.Errorf("backend %s does not support cache cleaning", e.officialBackend.Name())
}

// ListOrphans lists orphaned packages.
func (e *Engine) ListOrphans() ([]string, error) {
	if e.officialBackend == nil {
		return nil, ErrBackendNotFound
	}
	if m, ok := e.officialBackend.(Maintainer); ok {
		return m.ListOrphans()
	}
	return nil, fmt.Errorf("backend %s does not support listing orphans", e.officialBackend.Name())
}

// RemoveOrphans removes orphaned packages.
func (e *Engine) RemoveOrphans(pkgs []string) error {
	if e.officialBackend == nil {
		return ErrBackendNotFound
	}
	if m, ok := e.officialBackend.(Maintainer); ok {
		return m.RemoveOrphans(pkgs)
	}
	return fmt.Errorf("backend %s does not support removing orphans", e.officialBackend.Name())
}

// GetPackageFiles returns the files owned by a package.
func (e *Engine) GetPackageFiles(pkgName string) ([]string, error) {
	// 1. Try official backend
	if e.officialBackend != nil {
		if fp, ok := e.officialBackend.(FileProvider); ok {
			if files, err := fp.GetPackageFiles(pkgName); err == nil {
				return files, nil
			}
		}
	}

	// 2. Try other backends
	for _, b := range e.backends {
		if b == e.officialBackend {
			continue
		}
		if fp, ok := b.(FileProvider); ok {
			if files, err := fp.GetPackageFiles(pkgName); err == nil {
				return files, nil
			}
		}
	}

	return nil, ErrPackageNotFound
}

func (e *Engine) GetFileOwner(path string) (string, error) {
	// Try official backend
	if e.officialBackend != nil {
		if fp, ok := e.officialBackend.(FileProvider); ok {
			if owner, err := fp.GetFileOwner(path); err == nil {
				return owner, nil
			}
		}
	}
	// Try other backends
	for _, b := range e.backends {
		if fp, ok := b.(FileProvider); ok {
			if owner, err := fp.GetFileOwner(path); err == nil {
				return owner, nil
			}
		}
	}
	return "", fmt.Errorf("no package owns %s", path)
}

// SearchFiles searches for files in the package database.
func (e *Engine) SearchFiles(query string) ([]string, error) {
	supported := false
	var lastErr error
	// Try official backend
	if e.officialBackend != nil {
		if fp, ok := e.officialBackend.(FileProvider); ok {
			supported = true
			if files, err := fp.SearchFiles(query); err == nil {
				if files == nil {
					return []string{}, nil
				}
				return files, nil
			} else {
				lastErr = err
			}
		}
	}
	// Try other backends
	for _, b := range e.backends {
		if fp, ok := b.(FileProvider); ok {
			supported = true
			if files, err := fp.SearchFiles(query); err == nil {
				if files == nil {
					return []string{}, nil
				}
				return files, nil
			} else {
				lastErr = err
			}
		}
	}
	if supported {
		if lastErr != nil {
			return nil, lastErr
		}
		return []string{}, nil
	}
	return nil, fmt.Errorf("no backend supports file search")
}

// VerifyAll verifies all packages.
func (e *Engine) VerifyAll() (map[string][]string, error) {
	if e.officialBackend == nil {
		return nil, ErrBackendNotFound
	}
	if v, ok := e.officialBackend.(Verifier); ok {
		return v.VerifyAll()
	}
	return nil, fmt.Errorf("backend %s does not support verification", e.officialBackend.Name())
}

// VerifyPackage verifies a single package.
func (e *Engine) VerifyPackage(pkgName string) ([]string, error) {
	if e.officialBackend == nil {
		return nil, ErrBackendNotFound
	}
	if v, ok := e.officialBackend.(Verifier); ok {
		return v.VerifyPackage(pkgName)
	}
	return nil, fmt.Errorf("backend %s does not support verification", e.officialBackend.Name())
}

// Build builds a package from a directory.
func (e *Engine) Build(dir string, opts BuildOptions) error {
	if e.officialBackend == nil {
		return ErrBackendNotFound
	}
	if b, ok := e.officialBackend.(Builder); ok {
		return b.Build(dir, opts)
	}
	return fmt.Errorf("backend %s does not support building", e.officialBackend.Name())
}

// KeyringInit initializes the keyring.
func (e *Engine) KeyringInit() error {
	if e.officialBackend == nil {
		return ErrBackendNotFound
	}
	if k, ok := e.officialBackend.(KeyManager); ok {
		return k.InitKeyring()
	}
	return fmt.Errorf("backend %s does not support keyring management", e.officialBackend.Name())
}

// KeyringRefresh refreshes keys.
func (e *Engine) KeyringRefresh() error {
	if e.officialBackend == nil {
		return ErrBackendNotFound
	}
	if k, ok := e.officialBackend.(KeyManager); ok {
		return k.RefreshKeys()
	}
	return fmt.Errorf("backend %s does not support keyring management", e.officialBackend.Name())
}

// KeyringList lists keys.
func (e *Engine) KeyringList() ([]string, error) {
	if e.officialBackend == nil {
		return nil, ErrBackendNotFound
	}
	if k, ok := e.officialBackend.(KeyManager); ok {
		return k.ListKeys()
	}
	return nil, fmt.Errorf("backend %s does not support keyring management", e.officialBackend.Name())
}

// KeyringAdd adds a GPG key.
func (e *Engine) KeyringAdd(keyID string) error {
	if e.officialBackend == nil {
		return ErrBackendNotFound
	}
	if k, ok := e.officialBackend.(KeyManager); ok {
		return k.AddKey(keyID)
	}
	return fmt.Errorf("backend %s does not support keyring management", e.officialBackend.Name())
}

// KeyringRemove removes a GPG key.
func (e *Engine) KeyringRemove(keyID string) error {
	if e.officialBackend == nil {
		return ErrBackendNotFound
	}
	if k, ok := e.officialBackend.(KeyManager); ok {
		return k.RemoveKey(keyID)
	}
	return fmt.Errorf("backend %s does not support keyring management", e.officialBackend.Name())
}

// KeyringImport imports a GPG key from file.
func (e *Engine) KeyringImport(path string) error {
	if e.officialBackend == nil {
		return ErrBackendNotFound
	}
	if k, ok := e.officialBackend.(KeyManager); ok {
		return k.ImportKey(path)
	}
	return fmt.Errorf("backend %s does not support keyring management", e.officialBackend.Name())
}

// RepairLock removes the lock file.
func (e *Engine) RepairLock() error {
	if e.officialBackend == nil {
		return ErrBackendNotFound
	}
	if r, ok := e.officialBackend.(Repairer); ok {
		return r.RemoveLock()
	}
	return fmt.Errorf("backend %s does not support repair", e.officialBackend.Name())
}

// ListGroups lists available package groups.
func (e *Engine) ListGroups() ([]string, error) {
	if e.officialBackend == nil {
		return nil, ErrBackendNotFound
	}
	if g, ok := e.officialBackend.(GroupManager); ok {
		return g.ListGroups()
	}
	return nil, fmt.Errorf("backend %s does not support group management", e.officialBackend.Name())
}

// GetGroupPackages returns packages in a group.
func (e *Engine) GetGroupPackages(group string) ([]string, error) {
	if e.officialBackend == nil {
		return nil, ErrBackendNotFound
	}
	if g, ok := e.officialBackend.(GroupManager); ok {
		return g.GetGroupPackages(group)
	}
	return nil, fmt.Errorf("backend %s does not support group management", e.officialBackend.Name())
}

// SetInstallReason sets the install reason for a package.
func (e *Engine) SetInstallReason(pkg string, reason InstallReason) error {
	if e.officialBackend == nil {
		return ErrBackendNotFound
	}
	if dm, ok := e.officialBackend.(DatabaseManager); ok {
		return dm.SetInstallReason(pkg, reason)
	}
	return fmt.Errorf("backend %s does not support database management", e.officialBackend.Name())
}

// CheckDatabase checks the package database for internal consistency.
func (e *Engine) CheckDatabase() error {
	if e.officialBackend == nil {
		return ErrBackendNotFound
	}
	if dm, ok := e.officialBackend.(DatabaseManager); ok {
		return dm.CheckDatabase()
	}
	return fmt.Errorf("backend %s does not support database management", e.officialBackend.Name())
}

// ListExplicitPackages lists explicitly installed packages.
func (e *Engine) ListExplicitPackages() ([]string, error) {
	if e.officialBackend == nil {
		return nil, ErrBackendNotFound
	}
	if ex, ok := e.officialBackend.(Exporter); ok {
		return ex.ListExplicitPackages()
	}
	return nil, fmt.Errorf("backend %s does not support exporting", e.officialBackend.Name())
}

// Audit checks for security vulnerabilities.
func (e *Engine) Audit() ([]string, error) {
	if e.officialBackend == nil {
		return nil, ErrBackendNotFound
	}
	if s, ok := e.officialBackend.(SecurityScanner); ok {
		return s.Audit()
	}
	return nil, fmt.Errorf("backend %s does not support security auditing", e.officialBackend.Name())
}

// Diff returns pending update differences.
func (e *Engine) Diff() ([]PackageDiff, error) {
	if e.officialBackend == nil {
		return nil, ErrBackendNotFound
	}
	if d, ok := e.officialBackend.(Differ); ok {
		return d.Diff()
	}
	return nil, fmt.Errorf("backend %s does not support diffing", e.officialBackend.Name())
}
