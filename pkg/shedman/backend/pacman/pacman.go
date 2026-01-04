// Package pacman implements the OfficialBackend interface for Arch-based distributions.
package pacman

import (
	"bufio"
	"os"
	"os/exec"
	"strings"

	"github.com/theshedman/shedman/pkg/shedman/backend"
	"github.com/theshedman/shedman/pkg/shedman/config"
	"github.com/theshedman/shedman/pkg/shedman/pkgdb"
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

// Backend implements backend.OfficialBackend for Arch Linux
type Backend struct {
	executor   CommandExecutor
	binaryPath string
	sudoPath   string
}

// CommandExecutor allows mocking command execution in tests
type CommandExecutor interface {
	Run(name string, args ...string) error
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

// init registers the pacman backend factory
func init() {
	backend.RegisterBackend("pacman", func(cfg *config.BackendConfig) (backend.OfficialBackend, error) {
		// Try AlpmBackend first (native libalpm integration)
		alpmBackend, err := NewAlpmBackend()
		if err == nil {
			return alpmBackend, nil
		}

		// Fallback to shell-based Backend
		c := DefaultConfig()
		if cfg != nil && cfg.BinaryPath != "" {
			c.BinaryPath = cfg.BinaryPath
		}
		return NewWithConfig(c)
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
		return nil, backend.ErrBackendNotFound
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

// DistroFamily returns "arch"
func (b *Backend) DistroFamily() string {
	return "arch"
}

// IsAvailable checks if pacman is available
func (b *Backend) IsAvailable() bool {
	_, err := exec.LookPath(b.binaryPath)
	return err == nil
}

// Install installs packages from official repos
func (b *Backend) Install(pkgs []string, opts backend.InstallOptions) error {
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
func (b *Backend) Remove(pkgs []string, opts backend.RemoveOptions) error {
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
func (b *Backend) Upgrade(pkgs []string, opts backend.UpgradeOptions) error {
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
func (b *Backend) Search(query string) ([]pkgdb.PackageInfo, error) {
	output, err := b.executor.Output(b.binaryPath, "-Ss", query)
	if err != nil {
		return nil, err
	}

	return parsePacmanSearchOutput(string(output)), nil
}

// parsePacmanSearchOutput parses pacman -Ss output
func parsePacmanSearchOutput(output string) []pkgdb.PackageInfo {
	var results []pkgdb.PackageInfo
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

		results = append(results, pkgdb.PackageInfo{
			Name:        name,
			Version:     version,
			Description: desc,
			Source:      pkgdb.SourceOfficial,
		})
	}

	return results
}

// Info gets detailed package information
func (b *Backend) Info(pkgName string) (*pkgdb.PackageInfo, error) {
	output, err := b.executor.Output(b.binaryPath, "-Si", pkgName)
	if err != nil {
		return nil, backend.ErrPackageNotFound
	}

	return parsePacmanInfo(string(output)), nil
}

// parsePacmanInfo parses pacman -Si output
func parsePacmanInfo(output string) *pkgdb.PackageInfo {
	info := &pkgdb.PackageInfo{Source: pkgdb.SourceOfficial}
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
func (b *Backend) GetInstalledPackages() ([]pkgdb.PackageInfo, error) {
	output, err := b.executor.Output(b.binaryPath, "-Q")
	if err != nil {
		return nil, err
	}

	var packages []pkgdb.PackageInfo
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			packages = append(packages, pkgdb.PackageInfo{
				Name:    parts[0],
				Version: parts[1],
				Source:  pkgdb.SourceOfficial,
			})
		}
	}

	return packages, nil
}

// GetPackageFiles returns files owned by a package
func (b *Backend) GetPackageFiles(pkgName string) ([]string, error) {
	output, err := b.executor.Output(b.binaryPath, "-Ql", pkgName)
	if err != nil {
		return nil, backend.ErrPackageNotFound
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
	err := b.executor.Run(b.binaryPath, "-Q", pkgName)
	return err == nil
}

// InstallLocal installs a local package file
func (b *Backend) InstallLocal(path string, opts backend.InstallOptions) error {
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
