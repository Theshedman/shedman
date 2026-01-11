// Package backend provides the distribution-agnostic package manager interface.
// This allows shedman to work on any Linux distribution, not just Arch.
package backend

import (
	"github.com/theshedman/shedman/pkg/shedman/pkgdb"
)

// OfficialBackend is the distribution-specific package manager interface.
// Implementations exist for pacman (Arch), apt (Debian), dnf (Fedora), zypper (SUSE).
type OfficialBackend interface {
	// Name returns the backend name ("pacman", "apt", "dnf", "zypper")
	Name() string

	// DistroFamily returns the distro family ("arch", "debian", "fedora", "suse")
	DistroFamily() string

	// Install installs packages from official repos
	Install(pkgs []string, opts InstallOptions) error

	// Remove removes installed packages
	Remove(pkgs []string, opts RemoveOptions) error

	// Upgrade upgrades packages or entire system
	Upgrade(pkgs []string, opts UpgradeOptions) error

	// Sync refreshes package database
	Sync() error

	// Search searches for packages by name or description
	Search(query string) ([]pkgdb.PackageInfo, error)

	// Info gets detailed package information
	Info(pkgName string) (*pkgdb.PackageInfo, error)

	// GetInstalledPackages returns all installed packages
	GetInstalledPackages() ([]pkgdb.PackageInfo, error)

	// GetPackageFiles returns files owned by a package
	GetPackageFiles(pkgName string) ([]string, error)

	// IsInstalled checks if a package is installed
	IsInstalled(pkgName string) bool

	// InstallLocal installs a local package file (.pkg.tar.zst, .deb, .rpm)
	InstallLocal(path string, opts InstallOptions) error

	// IsAvailable checks if this backend is available on the system
	IsAvailable() bool
}
