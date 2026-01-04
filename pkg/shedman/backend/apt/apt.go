// Package apt implements the OfficialBackend interface for Debian-based distributions.
// This is a stub implementation - full functionality to be added when needed.
package apt

import (
	"errors"

	"github.com/theshedman/shedman/pkg/shedman/backend"
	"github.com/theshedman/shedman/pkg/shedman/config"
	"github.com/theshedman/shedman/pkg/shedman/pkgdb"
)

// ErrNotImplemented is returned for unimplemented methods
var ErrNotImplemented = errors.New("apt backend not yet implemented")

// Backend implements backend.OfficialBackend for Debian-based systems
type Backend struct{}

// init registers the apt backend factory
func init() {
	backend.RegisterBackend("apt", func(cfg *config.BackendConfig) (backend.OfficialBackend, error) {
		return New()
	})
}

// New creates a new APT backend
func New() (*Backend, error) {
	return &Backend{}, nil
}

// Name returns "apt"
func (b *Backend) Name() string {
	return "apt"
}

// DistroFamily returns "debian"
func (b *Backend) DistroFamily() string {
	return "debian"
}

// IsAvailable checks if apt is available
func (b *Backend) IsAvailable() bool {
	// TODO: Check for apt binary
	return false
}

// Install installs packages (stub)
func (b *Backend) Install(pkgs []string, opts backend.InstallOptions) error {
	return ErrNotImplemented
}

// Remove removes packages (stub)
func (b *Backend) Remove(pkgs []string, opts backend.RemoveOptions) error {
	return ErrNotImplemented
}

// Upgrade upgrades packages (stub)
func (b *Backend) Upgrade(pkgs []string, opts backend.UpgradeOptions) error {
	return ErrNotImplemented
}

// Sync refreshes package database (stub)
func (b *Backend) Sync() error {
	return ErrNotImplemented
}

// Search searches for packages (stub)
func (b *Backend) Search(query string) ([]pkgdb.PackageInfo, error) {
	return nil, ErrNotImplemented
}

// Info gets package info (stub)
func (b *Backend) Info(pkgName string) (*pkgdb.PackageInfo, error) {
	return nil, ErrNotImplemented
}

// GetInstalledPackages returns installed packages (stub)
func (b *Backend) GetInstalledPackages() ([]pkgdb.PackageInfo, error) {
	return nil, ErrNotImplemented
}

// GetPackageFiles returns files owned by a package (stub)
func (b *Backend) GetPackageFiles(pkgName string) ([]string, error) {
	return nil, ErrNotImplemented
}

// IsInstalled checks if a package is installed (stub)
func (b *Backend) IsInstalled(pkgName string) bool {
	return false
}

// InstallLocal installs a local .deb file (stub)
func (b *Backend) InstallLocal(path string, opts backend.InstallOptions) error {
	return ErrNotImplemented
}
