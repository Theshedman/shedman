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
}
