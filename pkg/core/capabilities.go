package core

// PackageManager is the capability to install and remove packages.
type PackageManager interface {
	Backend
	Install(pkgs []string, opts InstallOptions) error
	Remove(pkgs []string, opts RemoveOptions) error
	IsInstalled(pkgName string) bool
}

// Searchable is the capability to search for packages.
type Searchable interface {
	Search(query string) ([]PackageInfo, error)
}

// Informer is the capability to retrieve package information.
type Informer interface {
	Info(pkgName string) (*PackageInfo, error)
	GetInstalledPackages() ([]PackageInfo, error)
}

// Upgradable is the capability to upgrade the system or packages.
type Upgradable interface {
	Upgrade(pkgs []string, opts UpgradeOptions) error
}

// LocalInstaller is the capability to install packages from local files.
type LocalInstaller interface {
	InstallLocal(path string, opts InstallOptions) error
}

// FileProvider is the capability to list files owned by a package.
type FileProvider interface {
	GetPackageFiles(pkgName string) ([]string, error)
	GetFileOwner(path string) (string, error)
	SearchFiles(query string) ([]string, error)
}

// CleanOptions holds options for cache cleaning.
type CleanOptions struct {
	All  bool // Remove all files from cache (pacman -Scc)
	Keep int  // Number of recent versions to keep (paccache -rk)
}

// Maintainer is the capability to perform system maintenance.
type Maintainer interface {
	CleanCache(opts CleanOptions) error
	ListOrphans() ([]string, error)
	RemoveOrphans(pkgs []string) error
}

// Verifier is the capability to verify package integrity.
type Verifier interface {
	// VerifyAll returns map of pkgName -> list of errors
	VerifyAll() (map[string][]string, error)
	VerifyPackage(pkgName string) ([]string, error)
}

// Builder is the capability to build packages from source.
type Builder interface {
	Build(dir string, opts BuildOptions) error
}

// BuildOptions holds options for building.
type BuildOptions struct {
	Clean     bool
	Install   bool
	NoConfirm bool
	SynDeps   bool
}

// KeyManager is the capability to manage cryptographic keys.
type KeyManager interface {
	InitKeyring() error
	RefreshKeys() error
	ListKeys() ([]string, error)
	AddKey(keyID string) error
	RemoveKey(keyID string) error
	ImportKey(path string) error
}

// GroupManager is the capability to manage package groups.
type GroupManager interface {
	ListGroups() ([]string, error)
	GetGroupPackages(group string) ([]string, error)
}

// Repairer is the capability to repair the package database/system.
type Repairer interface {
	RemoveLock() error
}

// DatabaseManager is the capability to manage package database metadata.
type DatabaseManager interface {
	SetInstallReason(pkg string, reason InstallReason) error
	CheckDatabase() error
}

// Exporter is the capability to export package lists.
type Exporter interface {
	ListExplicitPackages() ([]string, error)
}

// Importer is the capability to import package lists.
type Importer interface {
	InstallFromList(path string) error
}

// SecurityScanner is the capability to audit installed packages for security vulnerabilities.
type SecurityScanner interface {
	Audit() ([]string, error)
}

// PackageDiff represents a pending update difference.
type PackageDiff struct {
	Name         string
	OldVersion   string
	NewVersion   string
	DownloadSize int64
	SizeDelta    int64
	CVEs         []string
	Pacnew       bool
}

// Differ is the capability to show pending update differences.
type Differ interface {
	Diff() ([]PackageDiff, error)
}

// Install installs the capability.
// InstallLocal installs a package from a local path.
