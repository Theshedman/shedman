package pacman

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	libalpm "github.com/Jguer/go-alpm/v2"
	"github.com/theshedman/shedman/internal/alpm"
	"github.com/theshedman/shedman/pkg/core"
)

// AlpmBackend implements core.OfficialBackend using libalpm directly.
// This is the preferred implementation when libalpm is available.
type AlpmBackend struct {
	handle   alpm.AlpmHandle
	executor CommandExecutor // Fallback for operations not supported by libalpm
	sudoPath string
}

// NewAlpmBackend creates a new backend using libalpm.
func NewAlpmBackend() (*AlpmBackend, error) {
	handle, err := alpm.NewRealAlpmHandle()
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
func NewAlpmBackendWithHandle(h alpm.AlpmHandle) *AlpmBackend {
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
func (b *AlpmBackend) runTransaction(flags libalpm.TransFlag, addPackages func() (int, error)) error {
	// Initialize transaction
	if err := b.handle.TransInit(flags); err != nil {
		return fmt.Errorf("failed to init transaction: %w", err)
	}
	defer func() { _ = b.handle.TransRelease() }()

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

// IsAvailable returns true if libalpm is initialized.
func (b *AlpmBackend) IsAvailable() bool {
	return b.handle != nil && b.handle.LocalDb() != nil
}

// Sync refreshes the package databases.
//
//	go-alpm v2 does not expose alpm_db_update(), so we use pacman binary
//
// This is the only operation that still requires the pacman binary.
func (b *AlpmBackend) Sync() error {
	return b.executor.Run(b.sudoPath, "pacman", "-Sy")
}

// Search searches for packages across all sync databases.
func (b *AlpmBackend) Search(query string) ([]core.PackageInfo, error) {
	var results []core.PackageInfo

	syncDbs := b.handle.SyncDbs()
	if syncDbs == nil {
		return nil, core.ErrBackendNotFound
	}

	err := syncDbs.ForEach(func(db alpm.AlpmDB) error {
		pkgList := db.Search([]string{query})
		if pkgList == nil {
			return nil
		}

		return pkgList.ForEach(func(pkg alpm.AlpmPackage) error {
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
func (b *AlpmBackend) Info(pkgName string) (*core.PackageInfo, error) {
	syncDbs := b.handle.SyncDbs()
	if syncDbs != nil {
		var found *core.PackageInfo
		_ = syncDbs.ForEach(func(db alpm.AlpmDB) error {

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

	return nil, core.ErrPackageNotFound
}

// GetInstalledPackages returns all installed packages.
func (b *AlpmBackend) GetInstalledPackages() ([]core.PackageInfo, error) {
	localDb := b.handle.LocalDb()
	if localDb == nil {
		return nil, core.ErrBackendNotFound
	}

	pkgCache := localDb.PkgCache()
	if pkgCache == nil {
		return nil, nil
	}

	var packages []core.PackageInfo
	err := pkgCache.ForEach(func(pkg alpm.AlpmPackage) error {
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
		return nil, core.ErrBackendNotFound
	}

	pkg := localDb.Pkg(pkgName)
	if pkg == nil {
		return nil, core.ErrPackageNotFound
	}

	files := pkg.Files()
	var result []string
	for _, f := range files {
		result = append(result, f.Name)
	}

	return result, nil
}

// InstallLocal installs a local package file from disk using pacman -U
func (b *AlpmBackend) InstallLocal(path string, opts core.InstallOptions) error {
	args := []string{"-U", path}

	// Add options
	if opts.NoConfirm {
		args = append(args, "--noconfirm")
	}
	if opts.AsDeps {
		args = append(args, "--asdeps")
	}
	if opts.Needed {
		args = append(args, "--needed")
	}
	if opts.Overwrite != "" {
		args = append(args, "--overwrite", "*")
	}

	execArgs := append([]string{"pacman"}, args...)
	return b.executor.Run(b.sudoPath, execArgs...)
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
func (b *AlpmBackend) Install(pkgs []string, opts core.InstallOptions) error {
	if len(pkgs) == 0 {
		return nil
	}

	// Build transaction flags
	flags := libalpm.TransFlag(0)
	if opts.AsDeps {
		flags |= libalpm.TransFlagAllDeps
	}
	if opts.DownloadOnly {
		flags |= libalpm.TransFlagDownloadOnly
	}

	return b.runTransaction(flags, func() (int, error) {
		syncDbs := b.handle.SyncDbs()
		if syncDbs == nil {
			return 0, core.ErrBackendNotFound
		}

		addedCount := 0
		for _, pkgName := range pkgs {
			if b.handle.IsIgnored(pkgName) {
				_, _ = fmt.Printf("Warning: %s is in IgnorePkg, skipping\n", pkgName)

				continue
			}

			var found bool
			var addErr error
			_ = syncDbs.ForEach(func(db alpm.AlpmDB) error {

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
				return addedCount, fmt.Errorf("%w: %s", core.ErrPackageNotFound, pkgName)
			}
		}
		return addedCount, nil
	})
}

// Remove removes packages using native libalpm transactions.
func (b *AlpmBackend) Remove(pkgs []string, opts core.RemoveOptions) error {
	if len(pkgs) == 0 {
		return nil
	}

	// Build transaction flags
	flags := libalpm.TransFlag(0)
	if opts.Cascade {
		flags |= libalpm.TransFlagCascade
	}
	if opts.NoSave {
		flags |= libalpm.TransFlagNoSave
	}
	if opts.Recursive {
		flags |= libalpm.TransFlagRecurse
	}

	return b.runTransaction(flags, func() (int, error) {
		localDb := b.handle.LocalDb()
		if localDb == nil {
			return 0, core.ErrBackendNotFound
		}

		removedCount := 0
		for _, pkgName := range pkgs {
			pkg := localDb.Pkg(pkgName)
			if pkg == nil {
				return removedCount, fmt.Errorf("%w: %s", core.ErrPackageNotFound, pkgName)
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
func (b *AlpmBackend) Upgrade(pkgs []string, opts core.UpgradeOptions) error {
	// Build transaction flags
	flags := libalpm.TransFlag(0)

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
func (b *AlpmBackend) addSpecificPackagesToUpgrade(pkgs []string, opts core.UpgradeOptions) (int, error) {
	syncDbs := b.handle.SyncDbs()
	if syncDbs == nil {
		return 0, core.ErrBackendNotFound
	}

	localDb := b.handle.LocalDb()
	addedCount := 0

	for _, pkgName := range pkgs {
		// Skip ignored packages
		if b.handle.IsIgnored(pkgName) {
			_, _ = fmt.Printf("Warning: %s is in IgnorePkg, skipping\n", pkgName)

			continue
		}

		var found bool
		var addErr error
		_ = syncDbs.ForEach(func(db alpm.AlpmDB) error {

			if found {
				return nil
			}
			pkg := db.Pkg(pkgName)
			if pkg != nil {
				if opts.Needed && localDb != nil {
					localPkg := localDb.Pkg(pkgName)
					if localPkg != nil && libalpm.VerCmp(pkg.Version(), localPkg.Version()) <= 0 {
						// Already up-to-date
						_, _ = fmt.Printf("%s is already up to date\n", pkgName)

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
			return addedCount, fmt.Errorf("%w: %s", core.ErrPackageNotFound, pkgName)
		}
	}
	return addedCount, nil
}

// addAllPackagesForUpgrade adds all installed packages with updates to the transaction
func (b *AlpmBackend) addAllPackagesForUpgrade() (int, error) {
	localDb := b.handle.LocalDb()
	syncDbs := b.handle.SyncDbs()
	if localDb == nil || syncDbs == nil {
		return 0, core.ErrBackendNotFound
	}

	upgradeCount := 0
	pkgCache := localDb.PkgCache()
	if pkgCache != nil {
		_ = pkgCache.ForEach(func(localPkg alpm.AlpmPackage) error {

			// Skip ignored packages
			if b.handle.IsIgnored(localPkg.Name()) {
				return nil
			}

			_ = syncDbs.ForEach(func(db alpm.AlpmDB) error {

				syncPkg := db.Pkg(localPkg.Name())
				if syncPkg != nil && libalpm.VerCmp(syncPkg.Version(), localPkg.Version()) > 0 {
					if err := b.handle.AddPkg(syncPkg); err != nil {
						// Log error but continue with other packages
						_, _ = fmt.Printf("Warning: failed to add %s to upgrade: %v\n", localPkg.Name(), err)

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

// GetFileOwner returns the owner of a file (via pacman -Qoq)
func (b *AlpmBackend) GetFileOwner(path string) (string, error) {
	output, err := b.executor.Output("pacman", "-Qoq", path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// CleanCache cleans the package cache (via pacman -Sc/Scc or paccache)
func (b *AlpmBackend) CleanCache(opts core.CleanOptions) error {
	if opts.Keep > 0 && !opts.All {
		return b.executor.Run(b.sudoPath, "paccache", "-rk", fmt.Sprintf("%d", opts.Keep))
	}

	args := []string{"-Sc"}
	if opts.All {
		args = []string{"-Scc"}
	}
	// Add noconfirm for automation if possible
	args = append(args, "--noconfirm")

	return b.executor.Run(b.sudoPath, append([]string{"pacman"}, args...)...)
}

// ListOrphans lists orphaned packages (via pacman -Qdtq)
func (b *AlpmBackend) ListOrphans() ([]string, error) {
	output, err := b.executor.Output("pacman", "-Qdtq")
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return []string{}, nil
		}
		return nil, err
	}
	return strings.Fields(string(output)), nil
}

// RemoveOrphans removes orphaned packages recursively (via pacman -Rns)
func (b *AlpmBackend) RemoveOrphans(pkgs []string) error {
	if len(pkgs) == 0 {
		return nil
	}
	args := append([]string{"-Rns"}, pkgs...)
	// Interactive
	return b.executor.Run(b.sudoPath, append([]string{"pacman"}, args...)...)
}

// VerifyAll verifies all packages (via pacman -Qkk)
func (b *AlpmBackend) VerifyAll() (map[string][]string, error) {
	output, err := b.executor.Output("pacman", "-Qkk")
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			return nil, err
		}
	}

	results := make(map[string][]string)
	lines := strings.Split(string(output), "\n")
	var pendingIssues []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if idx := strings.Index(line, ": "); idx > 0 {
			prefix := line[:idx]
			detail := line[idx+2:]

			if strings.Contains(detail, "total files") && strings.Contains(detail, "altered file") {
				pkgName := prefix

				if len(pendingIssues) > 0 {
					results[pkgName] = pendingIssues
					pendingIssues = nil
				}
				continue
			}
		}

		if strings.Contains(line, "mismatch") || strings.Contains(line, "missing") || strings.Contains(line, "altered file") {
			pendingIssues = append(pendingIssues, line)
		}
	}

	return results, nil
}

// VerifyPackage verifies a single package (via pacman -Qkk)
func (b *AlpmBackend) VerifyPackage(pkgName string) ([]string, error) {
	output, err := b.executor.Output("pacman", "-Qkk", pkgName)
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			return nil, err
		}
	}

	var issues []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		if strings.Contains(line, "mismatch") || strings.Contains(line, "missing") || strings.Contains(line, "altered file") {
			if !strings.Contains(line, "total files") {
				issues = append(issues, strings.TrimSpace(line))
			}
		}
	}

	return issues, nil
}

// Build builds a package from source using makepkg
func (b *AlpmBackend) Build(dir string, opts core.BuildOptions) error {
	args := []string{}

	if opts.Clean {
		args = append(args, "-c")
	}
	if opts.Install {
		args = append(args, "-i")
	}
	if opts.SynDeps {
		args = append(args, "-s")
	}
	if opts.NoConfirm {
		args = append(args, "--noconfirm")
	}

	cmd := exec.Command("makepkg", args...)
	if dir != "" && dir != "." {
		cmd.Dir = dir
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// Repairer implementation

// RemoveLock removes the pacman lock file
func (b *AlpmBackend) RemoveLock() error {
	lockFile := "/var/lib/pacman/db.lck"
	return b.executor.Run(b.sudoPath, "rm", "-f", lockFile)
}

// KeyManager implementation

// InitKeyring initializes the keyring (pacman-key --init and --populate)
func (b *AlpmBackend) InitKeyring() error {
	if err := b.executor.Run(b.sudoPath, "pacman-key", "--init"); err != nil {
		return err
	}
	// Typically populate is also needed for archlinux
	return b.executor.Run(b.sudoPath, "pacman-key", "--populate", "archlinux")
}

// RefreshKeys refreshes keys from keyservers
func (b *AlpmBackend) RefreshKeys() error {
	return b.executor.Run(b.sudoPath, "pacman-key", "--refresh-keys")
}

// ListKeys lists keys in the keyring
func (b *AlpmBackend) ListKeys() ([]string, error) {
	output, err := b.executor.Output("pacman-key", "--list-keys")
	if err != nil {
		outSudo, errSudo := b.executor.Output(b.sudoPath, "pacman-key", "--list-keys")
		if errSudo != nil {
			return nil, err
		}
		output = outSudo
	}

	return strings.Split(strings.TrimSpace(string(output)), "\n"), nil
}

// AddKey adds a key by ID (pacman-key --recv-keys)
func (b *AlpmBackend) AddKey(keyID string) error {
	return b.executor.Run(b.sudoPath, "pacman-key", "--recv-keys", keyID)
}

// RemoveKey removes a key by ID (pacman-key --delete)
func (b *AlpmBackend) RemoveKey(keyID string) error {
	return b.executor.Run(b.sudoPath, "pacman-key", "--delete", keyID)
}

// ImportKey imports a key from file (pacman-key --add)
func (b *AlpmBackend) ImportKey(path string) error {
	return b.executor.Run(b.sudoPath, "pacman-key", "--add", path)
}

// GroupManager implementation

// ListGroups lists all available package groups (via pacman -Sg)
func (b *AlpmBackend) ListGroups() ([]string, error) {
	output, err := b.executor.Output("pacman", "-Sg")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	var groups []string
	seen := make(map[string]bool)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) > 0 {
			groupName := parts[0]
			if !seen[groupName] {
				groups = append(groups, groupName)
				seen[groupName] = true
			}
		}
	}
	return groups, nil
}

// SearchFiles searches for files in the package database (via pacman -F)
func (b *AlpmBackend) SearchFiles(query string) ([]string, error) {
	output, err := b.executor.Output("pacman", "-F", query)
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return nil, nil // Not found
		}
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	var results []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			results = append(results, line)
		}
	}
	return results, nil
}
func (b *AlpmBackend) GetGroupPackages(group string) ([]string, error) {
	output, err := b.executor.Output("pacman", "-Sq", group)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	var pkgs []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			pkgs = append(pkgs, line)
		}
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("group %s not found or empty", group)
	}
	return pkgs, nil
}

// ListExplicitPackages lists explicitly installed packages.
func (b *AlpmBackend) ListExplicitPackages() ([]string, error) {
	output, err := b.executor.Output("pacman", "-Qqe")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result, nil
}

// Audit checks for security vulnerabilities.
func (b *AlpmBackend) Audit() ([]string, error) {
	if _, err := exec.LookPath("arch-audit"); err != nil {
		return nil, fmt.Errorf("arch-audit not found: please install 'arch-audit' to use this feature")
	}

	output, err := b.executor.Output("arch-audit")
	// arch-audit exits non-zero if vulnerabilities found
	outStr := string(output)
	if err != nil && outStr == "" {
		return nil, err
	}

	lines := strings.Split(outStr, "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result, nil
}

// Diff returns pending update differences.
func (b *AlpmBackend) Diff() ([]core.PackageDiff, error) {
	// 1. Get updates (pacman -Qu)
	output, err := b.executor.Output("pacman", "-Qu")
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return []core.PackageDiff{}, nil
		}
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	var diffs []core.PackageDiff

	// 2. Get CVEs map
	cveMap := make(map[string][]string)
	if _, err := exec.LookPath("arch-audit"); err == nil {
		// Use -f for machine readable output
		out, err := b.executor.Output("arch-audit", "-f", "%n|%c")
		if err == nil {
			for _, line := range strings.Split(string(out), "\n") {
				parts := strings.Split(line, "|")
				if len(parts) == 2 {
					cves := strings.Split(parts[1], ",")
					// Filter empty strings if any
					var cleanCVEs []string
					for _, c := range cves {
						if strings.TrimSpace(c) != "" {
							cleanCVEs = append(cleanCVEs, strings.TrimSpace(c))
						}
					}
					cveMap[parts[0]] = cleanCVEs
				}
			}
		}
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Format: name old -> new
		parts := strings.Fields(line)
		if len(parts) >= 4 {
			name := parts[0]
			oldVer := parts[1]
			newVer := parts[3]

			d := core.PackageDiff{
				Name:       name,
				OldVersion: oldVer,
				NewVersion: newVer,
				CVEs:       cveMap[name],
			}

			// Get sizes
			if out, err := b.executor.Output("pacman", "-Si", name); err == nil {
				d.DownloadSize = parsePacmanSize(string(out), "Download Size")
				newInstalledSize := parsePacmanSize(string(out), "Installed Size")

				if outQi, err := b.executor.Output("pacman", "-Qi", name); err == nil {
					oldInstalledSize := parsePacmanSize(string(outQi), "Installed Size")
					d.SizeDelta = newInstalledSize - oldInstalledSize
				}
			}

			// Check Pacnew potential (backup file modified)
			if out, err := b.executor.Output("pacman", "-Qii", name); err == nil {
				if strings.Contains(string(out), "MODIFIED") {
					d.Pacnew = true
				}
			}

			diffs = append(diffs, d)
		}
	}

	return diffs, nil
}

// DatabaseManager implementation

// SetInstallReason sets the install reason for a package via pacman -D
func (b *AlpmBackend) SetInstallReason(pkg string, reason core.InstallReason) error {
	args := []string{"-D"}
	if reason == core.InstallReasonDependency {
		args = append(args, "--asdeps")
	} else {
		args = append(args, "--asexplicit")
	}
	args = append(args, pkg)

	return b.executor.Run(b.sudoPath, append([]string{"pacman"}, args...)...)
}

// CheckDatabase checks the package database for internal consistency (via pacman -Dk)
func (b *AlpmBackend) CheckDatabase() error {
	return b.executor.Run(b.sudoPath, "pacman", "-Dk")
}
