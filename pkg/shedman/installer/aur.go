package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/theshedman/shedman/pkg/shedman/config"
)

// AURInstaller handles AUR package builds with full security features
type AURInstaller struct {
	executor        Executor
	cacheDir        string
	sandboxEnabled  bool
	cleanAfterBuild bool
}

// NewAURInstaller creates a new AURInstaller with default config
func NewAURInstaller() *AURInstaller {
	cfg := config.Default()
	return NewAURInstallerWithConfig(cfg)
}

// NewAURInstallerWithConfig creates a new AURInstaller with the given config
func NewAURInstallerWithConfig(cfg *config.Config) *AURInstaller {
	cacheDir := cfg.AUR.BuildDir
	if cacheDir == "" {
		home, _ := os.UserHomeDir()
		cacheDir = filepath.Join(home, ".cache", "shedman", "aur")
	}

	return &AURInstaller{
		executor:        DefaultExecutor,
		cacheDir:        cacheDir,
		sandboxEnabled:  true, // Sandbox enabled by default for security
		cleanAfterBuild: cfg.AUR.CleanAfterBuild,
	}
}

// GetCacheDir returns the cache directory
func (a *AURInstaller) GetCacheDir() string {
	return a.cacheDir
}

// SetExecutor sets a custom command executor (for testing)
func (a *AURInstaller) SetExecutor(exec Executor) {
	a.executor = exec
}

// SetCacheDir sets the cache directory (for testing)
func (a *AURInstaller) SetCacheDir(dir string) {
	a.cacheDir = dir
}

// SetSandboxEnabled enables/disables bubblewrap sandbox
func (a *AURInstaller) SetSandboxEnabled(enabled bool) {
	a.sandboxEnabled = enabled
}

// IsFirstTime checks if this is the first time building this package
func (a *AURInstaller) IsFirstTime(pkgName string) bool {
	gitDir := filepath.Join(a.cacheDir, pkgName, ".git")
	_, err := os.Stat(gitDir)
	return os.IsNotExist(err)
}

// Clone clones or updates the AUR package repository
func (a *AURInstaller) Clone(pkgName string) error {
	if !CommandExists("git") {
		return ErrGitNotFound
	}

	destDir := filepath.Join(a.cacheDir, pkgName)

	if a.IsFirstTime(pkgName) {
		// First time: git clone
		aurURL := fmt.Sprintf("https://aur.archlinux.org/%s.git", pkgName)
		if err := os.MkdirAll(a.cacheDir, DirPermissions); err != nil {
			return fmt.Errorf("failed to create cache directory: %w", err)
		}
		cmd := []string{"git", "clone", aurURL, destDir}
		return a.executor(cmd)
	}

	// Update: git fetch and reset
	fetchCmd := []string{"git", "-C", destDir, "fetch", "origin"}
	if err := a.executor(fetchCmd); err != nil {
		return err
	}

	resetCmd := []string{"git", "-C", destDir, "reset", "--hard", "origin/master"}
	return a.executor(resetCmd)
}

// GetPKGBUILD returns the PKGBUILD content for a package
func (a *AURInstaller) GetPKGBUILD(pkgName string) (string, error) {
	pkgbuildPath := filepath.Join(a.cacheDir, pkgName, "PKGBUILD")
	content, err := os.ReadFile(pkgbuildPath)
	if err != nil {
		return "", fmt.Errorf("failed to read PKGBUILD: %w", err)
	}
	return string(content), nil
}

// GetPKGBUILDDiff returns the diff of PKGBUILD changes since last build
func (a *AURInstaller) GetPKGBUILDDiff(pkgName string) (string, error) {
	pkgDir := filepath.Join(a.cacheDir, pkgName)

	// Use git diff to show changes
	cmd := []string{"git", "-C", pkgDir, "diff", "HEAD~1", "--", "PKGBUILD"}
	return "", a.executor(cmd)
}

// VerifyChecksums verifies source file checksums
func (a *AURInstaller) VerifyChecksums(pkgName string) error {
	if !CommandExists("makepkg") {
		return ErrMakepkgNotFound
	}

	pkgDir := filepath.Join(a.cacheDir, pkgName)
	cmd := []string{"sh", "-c", fmt.Sprintf("cd %s && makepkg --verifysource", pkgDir)}
	return a.executor(cmd)
}

// Build builds the AUR package using makepkg
func (a *AURInstaller) Build(pkgName string) error {
	if !CommandExists("makepkg") {
		return ErrMakepkgNotFound
	}

	pkgDir := filepath.Join(a.cacheDir, pkgName)

	if a.sandboxEnabled {
		if !CommandExists("bwrap") {
			return ErrBwrapNotFound
		}
		return a.buildWithSandbox(pkgDir)
	}
	return a.buildWithoutSandbox(pkgDir)
}

// buildWithSandbox builds using bubblewrap for isolation
func (a *AURInstaller) buildWithSandbox(pkgDir string) error {
	// Bubblewrap command with security restrictions:
	// - No network access (--unshare-net)
	// - Isolated home directory
	// - Read-only system directories
	// - Only build directory is writable
	cmd := []string{
		"bwrap",
		"--unshare-net",             // No network during build
		"--ro-bind", "/usr", "/usr", // Read-only /usr
		"--ro-bind", "/etc", "/etc", // Read-only /etc
		"--ro-bind", "/bin", "/bin", // Read-only /bin
		"--ro-bind", "/lib", "/lib", // Read-only /lib
		"--ro-bind", "/lib64", "/lib64", // Read-only /lib64
		"--tmpfs", "/tmp", // Temp directory
		"--bind", pkgDir, pkgDir, // Writable build directory
		"--chdir", pkgDir, // Change to build directory
		"--",
		"makepkg", "-s", "--noconfirm",
	}
	return a.executor(cmd)
}

// buildWithoutSandbox builds directly without isolation
func (a *AURInstaller) buildWithoutSandbox(pkgDir string) error {
	cmd := []string{"sh", "-c", fmt.Sprintf("cd %s && makepkg -s --noconfirm", pkgDir)}
	return a.executor(cmd)
}

// Install installs the built package using pacman
func (a *AURInstaller) Install(pkgName string) error {
	pkgDir := filepath.Join(a.cacheDir, pkgName)

	// Find the built package file
	pkgFile, err := a.findBuiltPackage(pkgDir)
	if err != nil {
		return err
	}

	cmd := []string{"sudo", "pacman", "-U", "--noconfirm", pkgFile}
	return a.executor(cmd)
}

// findBuiltPackage finds the .pkg.tar.zst file in the build directory
func (a *AURInstaller) findBuiltPackage(pkgDir string) (string, error) {
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, ".pkg.tar.zst") ||
			strings.HasSuffix(name, ".pkg.tar.xz") ||
			strings.HasSuffix(name, ".pkg.tar.gz") {
			return filepath.Join(pkgDir, name), nil
		}
	}

	return "", fmt.Errorf("no built package found in %s", pkgDir)
}
