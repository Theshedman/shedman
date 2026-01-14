// Package pacman implements the OfficialBackend interface for Arch-based distributions.
package pacman

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/pkg/core"
)

// Default binary paths
const (
	DefaultPacmanPath = "pacman"
	DefaultSudoPath   = "sudo"
)

// Config holds configuration for the pacman backend
type Config struct {
	BinaryPath string // Path to pacman binary (default: "pacman")
	SudoPath   string // Path to sudo binary (default: "sudo")
}

// DefaultConfig returns the default configuration
func DefaultConfig() Config {
	return Config{
		BinaryPath: DefaultPacmanPath,
		SudoPath:   DefaultSudoPath,
	}
}

// Backend implements core.OfficialBackend for Arch Linux
type Backend struct {
	executor   CommandExecutor
	binaryPath string
	sudoPath   string
}

// CommandExecutor allows mocking command execution in tests
type CommandExecutor interface {
	Run(name string, args ...string) error
	SilentRun(name string, args ...string) error
	Output(name string, args ...string) ([]byte, error)
}

// RealExecutor executes real system commands
type RealExecutor struct{}

func (r *RealExecutor) Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func (r *RealExecutor) Output(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

func (r *RealExecutor) SilentRun(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

// init registers the pacman backend factory
func init() {
	core.RegisterBackend("pacman", func(cfg *config.BackendConfig) (core.OfficialBackend, error) {
		// AlpmBackend is required - uses libalpm directly without shelling out to pacman
		alpmBackend, err := NewAlpmBackend()
		if err != nil {
			return nil, fmt.Errorf("libalpm backend required but not available: %w", err)
		}
		return alpmBackend, nil
	})
}

// New creates a new Pacman backend with default configuration
func New() (*Backend, error) {
	return NewWithConfig(DefaultConfig())
}

// NewWithConfig creates a new Pacman backend with custom configuration
func NewWithConfig(cfg Config) (*Backend, error) {
	if cfg.BinaryPath == "" {
		cfg.BinaryPath = DefaultPacmanPath
	}
	if cfg.SudoPath == "" {
		cfg.SudoPath = DefaultSudoPath
	}

	b := &Backend{
		executor:   &RealExecutor{},
		binaryPath: cfg.BinaryPath,
		sudoPath:   cfg.SudoPath,
	}

	if !b.IsAvailable() {
		return nil, core.ErrBackendNotFound
	}
	return b, nil
}

// NewWithExecutor creates a backend with a custom executor (for testing)
func NewWithExecutor(exec CommandExecutor) *Backend {
	return &Backend{
		executor:   exec,
		binaryPath: DefaultPacmanPath,
		sudoPath:   DefaultSudoPath,
	}
}

// Name returns "pacman"
func (b *Backend) Name() string {
	return "pacman"
}

// IsAvailable checks if pacman is available
func (b *Backend) IsAvailable() bool {
	_, err := exec.LookPath(b.binaryPath)
	return err == nil
}

// Install installs packages from official repos
func (b *Backend) Install(pkgs []string, opts core.InstallOptions) error {
	if len(pkgs) == 0 {
		return nil
	}

	args := []string{"-S"}

	if opts.Needed {
		args = append(args, "--needed")
	}
	if opts.AsDeps {
		args = append(args, "--asdeps")
	}
	if opts.AsExplicit {
		args = append(args, "--asexplicit")
	}
	if opts.NoConfirm {
		args = append(args, "--noconfirm")
	}
	if opts.DownloadOnly {
		args = append(args, "--downloadonly")
	}
	if opts.Overwrite != "" {
		args = append(args, "--overwrite", opts.Overwrite)
	}

	args = append(args, pkgs...)
	return b.executor.Run(b.sudoPath, append([]string{b.binaryPath}, args...)...)
}

// Remove removes installed packages
func (b *Backend) Remove(pkgs []string, opts core.RemoveOptions) error {
	if len(pkgs) == 0 {
		return nil
	}

	args := []string{"-R"}

	if opts.Cascade {
		args = append(args, "-c")
	}
	if opts.NoSave {
		args = append(args, "-n")
	}
	if opts.Recursive {
		args = append(args, "-s")
	}
	if opts.NoConfirm {
		args = append(args, "--noconfirm")
	}

	args = append(args, pkgs...)
	return b.executor.Run(b.sudoPath, append([]string{b.binaryPath}, args...)...)
}

// Upgrade upgrades packages or entire system
func (b *Backend) Upgrade(pkgs []string, opts core.UpgradeOptions) error {
	args := []string{"-S"}

	if opts.Refresh {
		args = append(args, "-y")
	}
	args = append(args, "-u")

	if opts.NoConfirm {
		args = append(args, "--noconfirm")
	}

	if len(pkgs) > 0 {
		args = append(args, pkgs...)
	}

	return b.executor.Run(b.sudoPath, append([]string{b.binaryPath}, args...)...)
}

// Sync refreshes package database
func (b *Backend) Sync() error {
	return b.executor.Run(b.sudoPath, b.binaryPath, "-Sy")
}

// Search searches for packages
func (b *Backend) Search(query string) ([]core.PackageInfo, error) {
	output, err := b.executor.Output(b.binaryPath, "-Ss", query)
	if err != nil {
		// pacman returns exit code 1 if no results found
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return []core.PackageInfo{}, nil
		}
		return nil, err
	}

	return parsePacmanSearchOutput(string(output)), nil
}

// parsePacmanSearchOutput parses pacman -Ss output
func parsePacmanSearchOutput(output string) []core.PackageInfo {
	var results []core.PackageInfo
	lines := strings.Split(output, "\n")

	for i := 0; i < len(lines)-1; i += 2 {
		header := lines[i]
		if header == "" {
			continue
		}

		// Format: repo/name version [installed]
		parts := strings.SplitN(header, " ", 2)
		if len(parts) < 2 {
			continue
		}

		repoName := parts[0]
		version := strings.Fields(parts[1])[0]

		// Extract name from repo/name
		nameParts := strings.SplitN(repoName, "/", 2)
		name := repoName
		if len(nameParts) == 2 {
			name = nameParts[1]
		}

		desc := ""
		if i+1 < len(lines) {
			desc = strings.TrimSpace(lines[i+1])
		}

		results = append(results, core.PackageInfo{
			Name:        name,
			Version:     version,
			Description: desc,
			Source:      core.SourceOfficial,
		})
	}

	return results
}

// Info gets detailed package information
func (b *Backend) Info(pkgName string) (*core.PackageInfo, error) {
	output, err := b.executor.Output(b.binaryPath, "-Si", pkgName)
	if err != nil {
		return nil, core.ErrPackageNotFound
	}

	return parsePacmanInfo(string(output)), nil
}

// parsePacmanInfo parses pacman -Si output
func parsePacmanInfo(output string) *core.PackageInfo {
	info := &core.PackageInfo{Source: core.SourceOfficial}
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(line, ":"); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])

			switch key {
			case "Name":
				info.Name = value
			case "Version":
				info.Version = value
			case "Description":
				info.Description = value
			case "Depends On":
				if value != "None" {
					info.Depends = strings.Fields(value)
				}
			case "Optional Deps":
				if value != "None" {
					info.OptDepends = []string{value}
				}
			case "Provides":
				if value != "None" {
					info.Provides = strings.Fields(value)
				}
			case "Conflicts With":
				if value != "None" {
					info.Conflicts = strings.Fields(value)
				}
			}
		}
	}

	return info
}

// GetInstalledPackages returns all installed packages
func (b *Backend) GetInstalledPackages() ([]core.PackageInfo, error) {
	output, err := b.executor.Output(b.binaryPath, "-Q")
	if err != nil {
		return nil, err
	}

	var packages []core.PackageInfo
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			packages = append(packages, core.PackageInfo{
				Name:    parts[0],
				Version: parts[1],
				Source:  core.SourceOfficial,
			})
		}
	}

	return packages, nil
}

// GetPackageFiles returns files owned by a package
func (b *Backend) GetPackageFiles(pkgName string) ([]string, error) {
	output, err := b.executor.Output(b.binaryPath, "-Ql", pkgName)
	if err != nil {
		return nil, core.ErrPackageNotFound
	}

	var files []string
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			files = append(files, parts[1])
		}
	}

	return files, nil
}

// IsInstalled checks if a package is installed
func (b *Backend) IsInstalled(pkgName string) bool {
	// Use SilentRun to avoid printing "error: package 'foo' was not found" to stderr
	err := b.executor.SilentRun(b.binaryPath, "-Q", pkgName)
	return err == nil
}

// InstallLocal installs a local package file
func (b *Backend) InstallLocal(path string, opts core.InstallOptions) error {
	args := []string{"-U"}

	if opts.Needed {
		args = append(args, "--needed")
	}
	if opts.AsDeps {
		args = append(args, "--asdeps")
	}
	if opts.AsExplicit {
		args = append(args, "--asexplicit")
	}
	if opts.NoConfirm {
		args = append(args, "--noconfirm")
	}

	args = append(args, path)
	return b.executor.Run(b.sudoPath, append([]string{b.binaryPath}, args...)...)
}

// GetFileOwner returns the owner of a file
func (b *Backend) GetFileOwner(path string) (string, error) {
	// Use -Qoq for quiet output (just package name) to avoid parsing issues
	output, err := b.executor.Output(b.binaryPath, "-Qoq", path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// CleanCache cleans the package cache
func (b *Backend) CleanCache(opts core.CleanOptions) error {
	if opts.Keep > 0 && !opts.All {
		return b.executor.Run(b.sudoPath, "paccache", "-rk", fmt.Sprintf("%d", opts.Keep))
	}

	args := []string{"-Sc"}
	if opts.All {
		args = []string{"-Scc"}
	}
	args = append(args, "--noconfirm")

	return b.executor.Run(b.sudoPath, append([]string{b.binaryPath}, args...)...)
}

// ListOrphans lists orphaned packages
func (b *Backend) ListOrphans() ([]string, error) {
	// -Qdtq: Query Deps(Orphans) Quiet
	output, err := b.executor.Output(b.binaryPath, "-Qdtq")
	if err != nil {
		// pacman returns exit code 1 if no orphans found
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return []string{}, nil
		}
		return nil, err
	}
	return strings.Fields(string(output)), nil
}

// RemoveOrphans removes orphaned packages recursively
func (b *Backend) RemoveOrphans(pkgs []string) error {
	if len(pkgs) == 0 {
		return nil
	}
	args := append([]string{"-Rns"}, pkgs...)
	return b.executor.Run(b.sudoPath, append([]string{b.binaryPath}, args...)...)
}

// VerifyAll verifies all packages
func (b *Backend) VerifyAll() (map[string][]string, error) {
	output, err := b.executor.Output(b.binaryPath, "-Qkk")
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

// VerifyPackage verifies a single package
func (b *Backend) VerifyPackage(pkgName string) ([]string, error) {
	output, err := b.executor.Output(b.binaryPath, "-Qkk", pkgName)
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
		if (strings.Contains(line, "mismatch") || strings.Contains(line, "missing") || strings.Contains(line, "altered file")) && !strings.Contains(line, "total files") {
			issues = append(issues, strings.TrimSpace(line))
		}
	}

	return issues, nil
}

// Build builds a package
func (b *Backend) Build(dir string, opts core.BuildOptions) error {
	// Not supporting build in pure pacman wrapper yet (requires makepkg)
	return fmt.Errorf("build not supported in this backend (use alpm backend)")
}

// KeyManager implementation

func (b *Backend) InitKeyring() error {
	// Shell out to pacman-key similarly
	return b.executor.Run(b.sudoPath, "pacman-key", "--init")
}

func (b *Backend) RefreshKeys() error {
	return b.executor.Run(b.sudoPath, "pacman-key", "--refresh-keys")
}

func (b *Backend) ListKeys() ([]string, error) {
	output, err := b.executor.Output(b.binaryPath+"-key", "--list-keys")
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimSpace(string(output)), "\n"), nil
}

func (b *Backend) AddKey(keyID string) error {
	return b.executor.Run(b.sudoPath, "pacman-key", "--recv-keys", keyID)
}

func (b *Backend) RemoveKey(keyID string) error {
	return b.executor.Run(b.sudoPath, "pacman-key", "--delete", keyID)
}

func (b *Backend) ImportKey(path string) error {
	return b.executor.Run(b.sudoPath, "pacman-key", "--add", path)
}

// Repairer implementation

func (b *Backend) RemoveLock() error {
	lockFile := "/var/lib/pacman/db.lck"
	return b.executor.Run(b.sudoPath, "rm", "-f", lockFile)
}

// SearchFiles searches for files in the package database (via pacman -F)
func (b *Backend) SearchFiles(query string) ([]string, error) {
	output, err := b.executor.Output(b.binaryPath, "-F", query)
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return nil, nil
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

// GroupManager implementation

func (b *Backend) ListGroups() ([]string, error) {
	output, err := b.executor.Output(b.binaryPath, "-Sg")
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

func (b *Backend) GetGroupPackages(group string) ([]string, error) {
	output, err := b.executor.Output(b.binaryPath, "-Sq", group)
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

// DatabaseManager implementation

func (b *Backend) SetInstallReason(pkg string, reason core.InstallReason) error {
	args := []string{"-D"}
	if reason == core.InstallReasonDependency {
		args = append(args, "--asdeps")
	} else {
		args = append(args, "--asexplicit")
	}
	args = append(args, pkg)

	return b.executor.Run(b.sudoPath, append([]string{b.binaryPath}, args...)...)
}

// CheckDatabase checks the package database for internal consistency (via pacman -Dk)
func (b *Backend) CheckDatabase() error {
	return b.executor.Run(b.sudoPath, b.binaryPath, "-Dk")
}

// ListExplicitPackages lists all explicitly installed packages (native + foreign).
func (b *Backend) ListExplicitPackages() ([]string, error) {
	// -Qqe lists all explicitly installed packages
	output, err := b.executor.Output(b.binaryPath, "-Qqe")
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

// Audit checks for security vulnerabilities using arch-audit.
func (b *Backend) Audit() ([]string, error) {
	// Check if arch-audit is available
	if _, err := exec.LookPath("arch-audit"); err != nil {
		return nil, fmt.Errorf("arch-audit not found: please install 'arch-audit' to use this feature")
	}

	// Run arch-audit
	output, err := b.executor.Output("arch-audit")
	// arch-audit exits non-zero if vulnerabilities found
	// Man page says: "Returns 0 if no vulnerable packages found, >0 otherwise."
	// So err != nil means we might have found vulnerabilities.

	outStr := string(output)
	if err != nil && outStr == "" {
		// Real error executing
		return nil, err
	}
	// If output has content, it lists vulnerabilities.

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
func (b *Backend) Diff() ([]core.PackageDiff, error) {
	// 1. Get updates (pacman -Qu)
	output, err := b.executor.Output(b.binaryPath, "-Qu")
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
			if out, err := b.executor.Output(b.binaryPath, "-Si", name); err == nil {
				d.DownloadSize = parsePacmanSize(string(out), "Download Size")
				newInstalledSize := parsePacmanSize(string(out), "Installed Size")

				if outQi, err := b.executor.Output(b.binaryPath, "-Qi", name); err == nil {
					oldInstalledSize := parsePacmanSize(string(outQi), "Installed Size")
					d.SizeDelta = newInstalledSize - oldInstalledSize
				}
			}

			// Check Pacnew potential (backup file modified)
			if out, err := b.executor.Output(b.binaryPath, "-Qii", name); err == nil {
				if strings.Contains(string(out), "MODIFIED") {
					d.Pacnew = true
				}
			}

			diffs = append(diffs, d)
		}
	}

	return diffs, nil
}
