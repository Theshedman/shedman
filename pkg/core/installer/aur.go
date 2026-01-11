package installer

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/pkg/backend"
	pacmanBackend "github.com/theshedman/shedman/pkg/backend/pacman"
)

const (
	DefaultAURURL = "https://aur.archlinux.org"
)

// AURInstaller handles AUR package builds with full security features
type AURInstaller struct {
	executor        Executor
	cacheDir        string
	sandboxEnabled  bool
	cleanAfterBuild bool
	backend         backend.OfficialBackend // Backend for local package installation
	aurURL          string                  // Configured AUR base URL
}

// NewAURInstaller creates a new AURInstaller with default config
func NewAURInstaller() *AURInstaller {
	cfg := config.Default()
	return NewAURInstallerWithConfig(cfg)
}

// NewAURInstallerWithConfig creates a new AURInstaller with the given config
// This auto-detects the backend; use NewAURInstallerWithBackend for explicit injection
func NewAURInstallerWithConfig(cfg *config.Config) *AURInstaller {
	// Auto-detect backend
	var b backend.OfficialBackend
	if pacmanBackend.IsAlpmAvailable() {
		b, _ = pacmanBackend.New()
	}
	return NewAURInstallerWithBackend(cfg, b)
}

// NewAURInstallerWithBackend creates a new AURInstaller with explicit backend injection
// This is the preferred constructor for production use and testing
func NewAURInstallerWithBackend(cfg *config.Config, b backend.OfficialBackend) *AURInstaller {
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
		backend:         b,
		aurURL:          cfg.Mirrors.AUR,
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

// SetBackend sets the backend for local package installation (for testing)
// Pass nil to use the fallback executor path
func (a *AURInstaller) SetBackend(b backend.OfficialBackend) {
	a.backend = b
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
		baseURL := a.aurURL
		if baseURL == "" {
			baseURL = DefaultAURURL
		}
		// Ensure simplified or full URL format is handled, typically <url>/<pkg>.git
		// If base URL ends with /rpc or similar, we might need adjustments,
		// but standard AUR mirror config implies base web URL.
		aurUrl := fmt.Sprintf("%s/%s.git", strings.TrimRight(baseURL, "/"), pkgName)

		if err := os.MkdirAll(a.cacheDir, DirPermissions); err != nil {
			return fmt.Errorf("failed to create cache directory: %w", err)
		}
		cmd := []string{"git", "clone", aurUrl, destDir}
		return a.executor("", cmd)
	}

	// Update: git fetch and reset
	fetchCmd := []string{"git", "-C", destDir, "fetch", "origin"}
	if err := a.executor("", fetchCmd); err != nil {
		return err
	}

	resetCmd := []string{"git", "-C", destDir, "reset", "--hard", "origin/master"}
	return a.executor("", resetCmd)
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

	// Check if we can get a diff (needs at least 2 commits)
	cmd := exec.Command("git", "-C", pkgDir, "rev-parse", "HEAD~1")
	if err := cmd.Run(); err != nil {
		// No previous commit, return full PKGBUILD instead
		return a.GetPKGBUILD(pkgName)
	}

	// Get the diff
	diffCmd := exec.Command("git", "-C", pkgDir, "diff", "HEAD~1", "--", "PKGBUILD")
	output, err := diffCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get PKGBUILD diff: %w", err)
	}

	if len(output) == 0 {
		return "No changes to PKGBUILD", nil
	}

	return string(output), nil
}

// VerifyChecksums verifies source file checksums
func (a *AURInstaller) VerifyChecksums(pkgName string) error {
	if !CommandExists("makepkg") {
		return ErrMakepkgNotFound
	}

	pkgDir := filepath.Join(a.cacheDir, pkgName)

	// Execute makepkg directly in the directory
	// Use executor with dir parameter
	cmd := []string{"makepkg", "--verifysource"}

	if err := a.executor(pkgDir, cmd); err != nil {
		return fmt.Errorf("makepkg --verifysource failed: %w", err)
	}
	return nil
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
		"--unshare-net", // No network during build
	}

	// Add read-only binds for directories that exist
	// On merged-usr systems, /bin /sbin /lib are symlinks to /usr counterparts
	optionalBinds := []string{
		"/usr",
		"/etc",
		"/bin",
		"/lib",
		"/lib64",
		"/sbin",
		"/var/cache/pacman",
	}

	for _, path := range optionalBinds {
		if sandboxPathExists(path) {
			cmd = append(cmd, "--ro-bind", path, path)
		}
	}

	// Required virtual filesystems
	cmd = append(cmd,
		"--dev", "/dev",
		"--proc", "/proc",
		"--tmpfs", "/tmp",
		"--tmpfs", "/home",
		"--bind", pkgDir, pkgDir,
		"--chdir", pkgDir,
		"--",
		"makepkg", "-s", "--noconfirm",
	)

	return a.executor(pkgDir, cmd)
}

// sandboxPathExists checks if a path exists for sandbox binding
func sandboxPathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// buildWithoutSandbox builds directly without isolation
func (a *AURInstaller) buildWithoutSandbox(pkgDir string) error {
	cmd := []string{"makepkg", "-s", "--noconfirm"}

	if err := a.executor(pkgDir, cmd); err != nil {
		return fmt.Errorf("makepkg build failed: %w", err)
	}
	return nil
}

// Install installs the built package using the backend
func (a *AURInstaller) Install(pkgName string) error {
	pkgDir := filepath.Join(a.cacheDir, pkgName)

	// Find the built package file
	_, err := a.findBuiltPackage(pkgDir)
	if err != nil {
		return err
	}

	// Backend is required - no fallback to pacman binary
	if a.backend == nil {
		return fmt.Errorf("no backend available for package installation")
	}

	_ = backend.InstallOptions{NoConfirm: true}
	return fmt.Errorf("local AUR package installation not yet implemented")
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

// Clean removes build artifacts from the package directory
func (a *AURInstaller) Clean(pkgName string) error {
	pkgDir := filepath.Join(a.cacheDir, pkgName)

	// Remove src/ and pkg/ directories created by makepkg
	srcDir := filepath.Join(pkgDir, "src")
	pkgSubDir := filepath.Join(pkgDir, "pkg")

	if err := os.RemoveAll(srcDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove src directory: %w", err)
	}
	if err := os.RemoveAll(pkgSubDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove pkg directory: %w", err)
	}

	// Remove built packages
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, ".pkg.tar.zst") ||
			strings.HasSuffix(name, ".pkg.tar.xz") ||
			strings.HasSuffix(name, ".pkg.tar.gz") {
			if err := os.Remove(filepath.Join(pkgDir, name)); err != nil {
				return err
			}
		}
	}

	return nil
}

// InstallFull performs the complete AUR installation workflow
func (a *AURInstaller) InstallFull(pkgName string, opts AUROptions) error {
	// 1. Clone or update repository
	if err := a.Clone(pkgName); err != nil {
		return fmt.Errorf("clone failed: %w", err)
	}

	// 2. Verify checksums if enabled
	if opts.VerifyChecksums {
		if err := a.VerifyChecksums(pkgName); err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}
	}

	// 3. Build the package
	if err := a.Build(pkgName); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	// 4. Install the package
	if err := a.Install(pkgName); err != nil {
		return fmt.Errorf("install failed: %w", err)
	}

	// 5. Clean if enabled
	if a.cleanAfterBuild {
		if err := a.Clean(pkgName); err != nil {
			// Don't fail on cleanup errors, just log
			fmt.Fprintf(os.Stderr, "Warning: cleanup failed: %v\n", err)
		}
	}

	return nil
}

// AUROptions holds options for AUR installation
type AUROptions struct {
	VerifyChecksums bool // Verify source checksums before build
	NoConfirm       bool // Skip confirmation prompts
	SkipPGPCheck    bool // Skip PGP signature verification
	FetchPGPKeys    bool // Auto-fetch PGP keys from keyserver
	Force           bool // Force reinstall even if up to date
}

// DefaultAUROptions returns sensible defaults
func DefaultAUROptions() AUROptions {
	return AUROptions{
		VerifyChecksums: true,
		FetchPGPKeys:    true,
	}
}

// AUROptionsFromConfig creates AUROptions from config settings
func AUROptionsFromConfig(cfg *config.Config) AUROptions {
	opts := DefaultAUROptions()
	if cfg != nil {
		opts.FetchPGPKeys = cfg.AUR.PGPFetch
	}
	return opts
}

// GetInstalledVersion returns the currently installed version of a package
// Returns empty string if not installed, error if backend fails
func (a *AURInstaller) GetInstalledVersion(pkgName string) (string, error) {
	if a.backend == nil {
		return "", fmt.Errorf("no backend available")
	}

	// Check if installed via backend
	if !a.backend.IsInstalled(pkgName) {
		return "", nil // Not installed
	}

	// Get package info to retrieve version
	info, err := a.backend.Info(pkgName)
	if err != nil {
		return "", fmt.Errorf("failed to get package info: %w", err)
	}

	if info == nil {
		return "", nil
	}

	return info.Version, nil
}

// NeedsUpdate checks if a package needs updating
func (a *AURInstaller) NeedsUpdate(pkgName, newVersion string) (bool, error) {
	installed, err := a.GetInstalledVersion(pkgName)
	if err != nil {
		return false, err
	}
	if installed == "" {
		return true, nil // Not installed, needs install
	}
	return installed != newVersion, nil
}

// AURStage represents stages of AUR installation
type AURStage int

const (
	AURStageClone AURStage = iota
	AURStageFetchPGP
	AURStageVerify
	AURStageBuild
	AURStageInstall
	AURStageClean
	AURStageComplete
)

// String returns a human-readable stage name
func (s AURStage) String() string {
	switch s {
	case AURStageClone:
		return "Cloning"
	case AURStageFetchPGP:
		return "Fetching PGP keys"
	case AURStageVerify:
		return "Verifying checksums"
	case AURStageBuild:
		return "Building"
	case AURStageInstall:
		return "Installing"
	case AURStageClean:
		return "Cleaning up"
	case AURStageComplete:
		return "Complete"
	default:
		return "Unknown"
	}
}

// AURProgressCallback is called with progress updates during AUR installation
type AURProgressCallback func(stage AURStage, packageName string, message string)

// FetchPGPKeys extracts validpgpkeys from PKGBUILD and fetches them
func (a *AURInstaller) FetchPGPKeys(pkgName string) error {
	pkgbuild, err := a.GetPKGBUILD(pkgName)
	if err != nil {
		return err
	}

	// Parse validpgpkeys array from PKGBUILD
	keys := extractPGPKeys(pkgbuild)
	if len(keys) == 0 {
		return nil // No keys to fetch
	}

	// Fetch each key from keyserver
	for _, key := range keys {
		cmd := []string{"gpg", "--keyserver", "keyserver.ubuntu.com", "--recv-keys", key}
		if err := a.executor("", cmd); err != nil {
			// Try another keyserver
			cmd = []string{"gpg", "--keyserver", "keys.openpgp.org", "--recv-keys", key}
			if err := a.executor("", cmd); err != nil {
				return fmt.Errorf("failed to fetch PGP key %s: %w", key, err)
			}
		}
	}

	return nil
}

// extractPGPKeys parses validpgpkeys from PKGBUILD content
func extractPGPKeys(pkgbuild string) []string {
	// Match validpgpkeys=('key1' 'key2' ...) or validpgpkeys=("key1" "key2" ...)
	re := regexp.MustCompile(`validpgpkeys=\([^)]+\)`)
	match := re.FindString(pkgbuild)
	if match == "" {
		return nil
	}

	// Extract individual keys
	keyRe := regexp.MustCompile(`['"]([A-Fa-f0-9]{8,})['"]`)
	matches := keyRe.FindAllStringSubmatch(match, -1)

	keys := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) > 1 {
			keys = append(keys, m[1])
		}
	}

	return keys
}

// InstallFullWithProgress performs the complete AUR installation with progress callbacks
func (a *AURInstaller) InstallFullWithProgress(pkgName string, opts AUROptions, callback AURProgressCallback) error {
	report := func(stage AURStage, msg string) {
		if callback != nil {
			callback(stage, pkgName, msg)
		}
	}

	// 1. Clone or update repository
	report(AURStageClone, "Cloning/updating AUR repository...")
	if err := a.Clone(pkgName); err != nil {
		return fmt.Errorf("clone failed: %w", err)
	}

	// 2. Fetch PGP keys if enabled
	if opts.FetchPGPKeys {
		report(AURStageFetchPGP, "Fetching PGP keys...")
		if err := a.FetchPGPKeys(pkgName); err != nil {
			// Log but don't fail - keys might already exist
			fmt.Fprintf(os.Stderr, "Warning: PGP key fetch: %v\n", err)
		}
	}

	// 3. Verify checksums if enabled
	if opts.VerifyChecksums {
		report(AURStageVerify, "Verifying source checksums...")
		if err := a.VerifyChecksums(pkgName); err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}
	}

	// 4. Build the package
	report(AURStageBuild, "Building package...")
	if err := a.Build(pkgName); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	// 5. Install the package
	report(AURStageInstall, "Installing package...")
	if err := a.Install(pkgName); err != nil {
		return fmt.Errorf("install failed: %w", err)
	}

	// 6. Clean if enabled
	if a.cleanAfterBuild {
		report(AURStageClean, "Cleaning build artifacts...")
		if err := a.Clean(pkgName); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: cleanup failed: %v\n", err)
		}
	}

	report(AURStageComplete, "Installation complete")
	return nil
}

// GetPKGBUILDVersion extracts pkgver from PKGBUILD
func (a *AURInstaller) GetPKGBUILDVersion(pkgName string) (string, error) {
	pkgbuild, err := a.GetPKGBUILD(pkgName)
	if err != nil {
		return "", err
	}

	// Parse pkgver= line
	scanner := bufio.NewScanner(strings.NewReader(pkgbuild))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "pkgver=") {
			version := strings.TrimPrefix(line, "pkgver=")
			version = strings.Trim(version, "'\"")
			return version, nil
		}
	}

	return "", fmt.Errorf("pkgver not found in PKGBUILD")
}
