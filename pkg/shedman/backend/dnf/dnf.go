// Package dnf implements the OfficialBackend interface for Fedora-based distributions.
// This is a stub implementation - full functionality to be added when needed.
package dnf

import (
	"errors"

	"github.com/theshedman/shedman/pkg/shedman/backend"
	"github.com/theshedman/shedman/pkg/shedman/config"
	"github.com/theshedman/shedman/pkg/shedman/pkgdb"
)

// ErrNotImplemented is returned for unimplemented methods
var ErrNotImplemented = errors.New("dnf backend not yet implemented")

// Backend implements backend.OfficialBackend for Fedora-based systems
type Backend struct{}

// init registers the dnf backend factory
func init() {
	backend.RegisterBackend("dnf", func(cfg *config.BackendConfig) (backend.OfficialBackend, error) {
		return New()
	})
}

// New creates a new DNF backend
func New() (*Backend, error) {
	return &Backend{}, nil
}

// Name returns "dnf"
func (b *Backend) Name() string {
	return "dnf"
}

// DistroFamily returns "fedora"
func (b *Backend) DistroFamily() string {
	return "fedora"
}

// IsAvailable checks if dnf is available
func (b *Backend) IsAvailable() bool {
	// TODO: Check for dnf binary
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

// InstallLocal installs a local .rpm file (stub)
func (b *Backend) InstallLocal(path string, opts backend.InstallOptions) error {
	return ErrNotImplemented
}
