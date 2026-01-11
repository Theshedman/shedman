package pacman

import (
	"fmt"

	"github.com/Jguer/go-alpm/v2"
	"github.com/theshedman/shedman/pkg/backend"
	"github.com/theshedman/shedman/pkg/core/pkgdb"
)

// AlpmBackend implements backend.OfficialBackend using libalpm directly.
// This is the preferred implementation when libalpm is available.
type AlpmBackend struct {
	handle   AlpmHandle
	executor CommandExecutor // Fallback for operations not supported by libalpm
	sudoPath string
}

// NewAlpmBackend creates a new backend using libalpm.
func NewAlpmBackend() (*AlpmBackend, error) {
	handle, err := NewRealAlpmHandle()
	if err != nil {
		return nil, err
	}

	return &AlpmBackend{
		handle:   handle,
		executor: &RealExecutor{},
		sudoPath: DefaultSudoPath,
	}, nil
}

// NewAlpmBackendWithHandle creates a backend with a custom handle (for testing).
func NewAlpmBackendWithHandle(h AlpmHandle) *AlpmBackend {
	return &AlpmBackend{
		handle:   h,
		executor: &RealExecutor{},
		sudoPath: DefaultSudoPath,
	}
}

// Close releases the libalpm handle.
func (b *AlpmBackend) Close() error {
	if b.handle != nil {
		return b.handle.Release()
	}
	return nil
}

// runTransaction executes a libalpm transaction with proper lifecycle management.
// The addPackages callback should add packages to the transaction and return the count.
// Returns nil if no packages were added (nothing to do).
func (b *AlpmBackend) runTransaction(flags alpm.TransFlag, addPackages func() (int, error)) error {
	// Initialize transaction
	if err := b.handle.TransInit(flags); err != nil {
		return fmt.Errorf("failed to init transaction: %w", err)
	}
	defer b.handle.TransRelease()

	// Add packages to transaction
	count, err := addPackages()
	if err != nil {
		return err
	}

	// Nothing to do if no packages added
	if count == 0 {
		return nil
	}

	// Prepare transaction
	if err := b.handle.TransPrepare(); err != nil {
		return fmt.Errorf("failed to prepare transaction: %w", err)
	}

	// Commit transaction
	if err := b.handle.TransCommit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Name returns "pacman".
func (b *AlpmBackend) Name() string {
	return "pacman"
}

// DistroFamily returns "arch".
func (b *AlpmBackend) DistroFamily() string {
	return "arch"
}

// IsAvailable returns true if libalpm is initialized.
func (b *AlpmBackend) IsAvailable() bool {
	return b.handle != nil && b.handle.LocalDb() != nil
}

// Sync refreshes the package databases.
// NOTE: go-alpm v2 does not expose alpm_db_update(), so we use pacman binary.
// This is the only operation that still requires the pacman binary.
func (b *AlpmBackend) Sync() error {
	return b.executor.Run(b.sudoPath, "pacman", "-Sy")
}

// Search searches for packages across all sync databases.
func (b *AlpmBackend) Search(query string) ([]pkgdb.PackageInfo, error) {
	var results []pkgdb.PackageInfo

	syncDbs := b.handle.SyncDbs()
	if syncDbs == nil {
		return nil, backend.ErrBackendNotFound
	}

	err := syncDbs.ForEach(func(db AlpmDB) error {
		pkgList := db.Search([]string{query})
		if pkgList == nil {
			return nil
		}

		return pkgList.ForEach(func(pkg AlpmPackage) error {
			results = append(results, PackageToInfo(pkg))
			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	return results, nil
}

// Info returns detailed information about a package.
func (b *AlpmBackend) Info(pkgName string) (*pkgdb.PackageInfo, error) {
	// Check sync databases first
	syncDbs := b.handle.SyncDbs()
	if syncDbs != nil {
		var found *pkgdb.PackageInfo
		syncDbs.ForEach(func(db AlpmDB) error {
			if found != nil {
				return nil // Already found
			}
			pkg := db.Pkg(pkgName)
			if pkg != nil {
				info := PackageToInfo(pkg)
				found = &info
			}
			return nil
		})
		if found != nil {
			return found, nil
		}
	}

	return nil, backend.ErrPackageNotFound
}

// GetInstalledPackages returns all installed packages.
func (b *AlpmBackend) GetInstalledPackages() ([]pkgdb.PackageInfo, error) {
	localDb := b.handle.LocalDb()
	if localDb == nil {
		return nil, backend.ErrBackendNotFound
	}

	pkgCache := localDb.PkgCache()
	if pkgCache == nil {
		return nil, nil
	}

	var packages []pkgdb.PackageInfo
	err := pkgCache.ForEach(func(pkg AlpmPackage) error {
		packages = append(packages, PackageToInfo(pkg))
		return nil
	})

	if err != nil {
		return nil, err
	}

	return packages, nil
}

// GetPackageFiles returns the list of files owned by a package.
func (b *AlpmBackend) GetPackageFiles(pkgName string) ([]string, error) {
	localDb := b.handle.LocalDb()
	if localDb == nil {
		return nil, backend.ErrBackendNotFound
	}

	pkg := localDb.Pkg(pkgName)
	if pkg == nil {
		return nil, backend.ErrPackageNotFound
	}

	files := pkg.Files()
	var result []string
	for _, f := range files {
		result = append(result, f.Name)
	}

	return result, nil
}

// IsInstalled checks if a package is installed.
func (b *AlpmBackend) IsInstalled(pkgName string) bool {
	localDb := b.handle.LocalDb()
	if localDb == nil {
		return false
	}

	return localDb.Pkg(pkgName) != nil
}

// Install installs packages using native libalpm transactions.
func (b *AlpmBackend) Install(pkgs []string, opts backend.InstallOptions) error {
	if len(pkgs) == 0 {
		return nil
	}

	// Build transaction flags
	flags := alpm.TransFlag(0)
	if opts.AsDeps {
		flags |= alpm.TransFlagAllDeps
	}
	if opts.DownloadOnly {
		flags |= alpm.TransFlagDownloadOnly
	}

	return b.runTransaction(flags, func() (int, error) {
		syncDbs := b.handle.SyncDbs()
		if syncDbs == nil {
			return 0, backend.ErrBackendNotFound
		}

		addedCount := 0
		for _, pkgName := range pkgs {
			// Check if package is ignored
			if b.handle.IsIgnored(pkgName) {
				fmt.Printf("Warning: %s is in IgnorePkg, skipping\n", pkgName)
				continue
			}

			var found bool
			var addErr error
			syncDbs.ForEach(func(db AlpmDB) error {
				if found {
					return nil
				}
				pkg := db.Pkg(pkgName)
				if pkg != nil {
					if opts.Needed && b.IsInstalled(pkgName) {
						// Skip if already installed and --needed
						found = true
						return nil
					}
					if err := b.handle.AddPkg(pkg); err != nil {
						addErr = err
						return err
					}
					found = true
					addedCount++
				}
				return nil
			})
			if addErr != nil {
				return addedCount, addErr
			}
			if !found {
				return addedCount, fmt.Errorf("%w: %s", backend.ErrPackageNotFound, pkgName)
			}
		}
		return addedCount, nil
	})
}

// Remove removes packages using native libalpm transactions.
func (b *AlpmBackend) Remove(pkgs []string, opts backend.RemoveOptions) error {
	if len(pkgs) == 0 {
		return nil
	}

	// Build transaction flags
	flags := alpm.TransFlag(0)
	if opts.Cascade {
		flags |= alpm.TransFlagCascade
	}
	if opts.NoSave {
		flags |= alpm.TransFlagNoSave
	}
	if opts.Recursive {
		flags |= alpm.TransFlagRecurse
	}

	return b.runTransaction(flags, func() (int, error) {
		localDb := b.handle.LocalDb()
		if localDb == nil {
			return 0, backend.ErrBackendNotFound
		}

		removedCount := 0
		for _, pkgName := range pkgs {
			pkg := localDb.Pkg(pkgName)
			if pkg == nil {
				return removedCount, fmt.Errorf("%w: %s", backend.ErrPackageNotFound, pkgName)
			}
			if err := b.handle.RemovePkg(pkg); err != nil {
				return removedCount, fmt.Errorf("failed to add package for removal: %w", err)
			}
			removedCount++
		}
		return removedCount, nil
	})
}

// Upgrade upgrades packages using native libalpm.
func (b *AlpmBackend) Upgrade(pkgs []string, opts backend.UpgradeOptions) error {
	// Build transaction flags
	flags := alpm.TransFlag(0)

	// Sync databases if requested
	if opts.Refresh {
		if err := b.Sync(); err != nil {
			return fmt.Errorf("failed to sync databases: %w", err)
		}
	}

	return b.runTransaction(flags, func() (int, error) {
		// If specific packages, find their newer versions
		if len(pkgs) > 0 {
			return b.addSpecificPackagesToUpgrade(pkgs, opts)
		}
		// Full system upgrade
		return b.addAllPackagesForUpgrade()
	})
}

// addSpecificPackagesToUpgrade adds specific packages to the upgrade transaction
func (b *AlpmBackend) addSpecificPackagesToUpgrade(pkgs []string, opts backend.UpgradeOptions) (int, error) {
	syncDbs := b.handle.SyncDbs()
	if syncDbs == nil {
		return 0, backend.ErrBackendNotFound
	}

	localDb := b.handle.LocalDb()
	addedCount := 0

	for _, pkgName := range pkgs {
		// Skip ignored packages
		if b.handle.IsIgnored(pkgName) {
			fmt.Printf("Warning: %s is in IgnorePkg, skipping\n", pkgName)
			continue
		}

		var found bool
		var addErr error
		syncDbs.ForEach(func(db AlpmDB) error {
			if found {
				return nil
			}
			pkg := db.Pkg(pkgName)
			if pkg != nil {
				// Check if --needed and already up-to-date
				if opts.Needed && localDb != nil {
					localPkg := localDb.Pkg(pkgName)
					if localPkg != nil && alpm.VerCmp(pkg.Version(), localPkg.Version()) <= 0 {
						// Already up-to-date
						fmt.Printf("%s is already up to date\n", pkgName)
						found = true
						return nil
					}
				}

				if err := b.handle.AddPkg(pkg); err != nil {
					addErr = err
					return err
				}
				found = true
				addedCount++
			}
			return nil
		})
		if addErr != nil {
			return addedCount, addErr
		}
		if !found {
			return addedCount, fmt.Errorf("%w: %s", backend.ErrPackageNotFound, pkgName)
		}
	}
	return addedCount, nil
}

// addAllPackagesForUpgrade adds all installed packages with updates to the transaction
func (b *AlpmBackend) addAllPackagesForUpgrade() (int, error) {
	localDb := b.handle.LocalDb()
	syncDbs := b.handle.SyncDbs()
	if localDb == nil || syncDbs == nil {
		return 0, backend.ErrBackendNotFound
	}

	upgradeCount := 0
	pkgCache := localDb.PkgCache()
	if pkgCache != nil {
		pkgCache.ForEach(func(localPkg AlpmPackage) error {
			// Skip ignored packages
			if b.handle.IsIgnored(localPkg.Name()) {
				return nil
			}

			// Check for newer version in sync DBs
			syncDbs.ForEach(func(db AlpmDB) error {
				syncPkg := db.Pkg(localPkg.Name())
				if syncPkg != nil && alpm.VerCmp(syncPkg.Version(), localPkg.Version()) > 0 {
					if err := b.handle.AddPkg(syncPkg); err != nil {
						// Log error but continue with other packages
						fmt.Printf("Warning: failed to add %s to upgrade: %v\n", localPkg.Name(), err)
					} else {
						upgradeCount++
					}
				}
				return nil
			})
			return nil
		})
	}

	return upgradeCount, nil
}

// // InstallLocal installs a local package file by converting to .shed format.
// // Supports .pkg.tar.zst, .pkg.tar.xz, .pkg.tar.gz, and .shed formats.
// func (b *AlpmBackend) InstallLocal(path string, opts backend.InstallOptions) error {
// 	// Detect package format
// 	format := convert.DetectPackageFormat(path)
// 
// 	var shedPath string
// 	var err error
// 
// 	switch format {
// 	case "shed":
// 		// Already .shed format, use directly
// 		shedPath = path
// 
// 	case "pacman-zst", "pacman-xz", "pacman-gz":
// 		// Convert pacman package to .shed format
// 		converter := convert.NewPackageConverter()
// 		shedPath, err = converter.ConvertPacmanToShed(path)
// 		if err != nil {
// 			return fmt.Errorf("failed to convert package: %w", err)
// 		}
// 
// 	default:
// 		return fmt.Errorf("unsupported package format: %s", format)
// 	}
// 
// 	// Install the .shed package
// 	if err := convert.InstallShed(shedPath); err != nil {
// 		return fmt.Errorf("shed installation failed: %w", err)
// 	}
// 
// 	return nil
// }
