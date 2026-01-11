package installer

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"github.com/theshedman/shedman/pkg/shedman/output"
)

// Common errors
var (
	ErrCommandNotFound = errors.New("required command not found")
	ErrGPGNotFound     = errors.New("gpg is required for signature verification but not found in PATH")
	ErrTarNotFound     = errors.New("tar is required for extraction but not found in PATH")
	ErrGitNotFound     = errors.New("git is required for AUR operations but not found in PATH")
	ErrMakepkgNotFound = errors.New("makepkg is required for AUR builds but not found in PATH")
	ErrBwrapNotFound   = errors.New("bubblewrap (bwrap) is required for sandbox builds but not found in PATH")
)

// File permission constants
const (
	DirPermissions  os.FileMode = 0755 // rwxr-xr-x for directories
	FilePermissions os.FileMode = 0644 // rw-r--r-- for regular files
	ExecPermissions os.FileMode = 0755 // rwxr-xr-x for executables
)

// CommandExists checks if a command is available in PATH
func CommandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// ShedManifest represents the manifest.toml in a .shed package
type ShedManifest struct {
	Name        string   `toml:"name"`
	Version     string   `toml:"version"`
	Description string   `toml:"description"`
	Depends     []string `toml:"depends"`
	Provides    []string `toml:"provides"`
	Conflicts   []string `toml:"conflicts"`
	Files       []string `toml:"files"` // List of installed files for removal
}

// ShedInstaller handles .shed package installation
type ShedInstaller struct {
	executor Executor
	cacheDir string
}

// NewShedInstaller creates a new ShedInstaller with default cache directory
func NewShedInstaller() *ShedInstaller {
	home, _ := os.UserHomeDir()
	return &ShedInstaller{
		executor: DefaultExecutor,
		cacheDir: filepath.Join(home, ".cache", "shedman", "shed"),
	}
}

// NewShedInstallerWithCacheDir creates a new ShedInstaller with a custom cache directory
func NewShedInstallerWithCacheDir(cacheDir string) *ShedInstaller {
	if cacheDir == "" {
		home, _ := os.UserHomeDir()
		cacheDir = filepath.Join(home, ".cache", "shedman", "shed")
	}
	return &ShedInstaller{
		executor: DefaultExecutor,
		cacheDir: cacheDir,
	}
}

// GetCacheDir returns the cache directory
func (s *ShedInstaller) GetCacheDir() string {
	return s.cacheDir
}

// SetExecutor sets a custom command executor (for testing)
func (s *ShedInstaller) SetExecutor(exec Executor) {
	s.executor = exec
}

// SetCacheDir sets the cache directory (for testing)
func (s *ShedInstaller) SetCacheDir(dir string) {
	s.cacheDir = dir
}

// Extract extracts a .shed package to the destination directory
func (s *ShedInstaller) Extract(shedFile, destDir string) error {
	if !CommandExists("tar") {
		return ErrTarNotFound
	}

	if err := os.MkdirAll(destDir, DirPermissions); err != nil {
		return fmt.Errorf("failed to create extraction directory: %w", err)
	}

	cmd := []string{"tar", "-xf", shedFile, "-C", destDir}
	return s.executor("", cmd)
}

// ReadManifest reads and parses the manifest.toml from an extracted package
func (s *ShedInstaller) ReadManifest(pkgDir string) (*ShedManifest, error) {
	manifestPath := filepath.Join(pkgDir, "manifest.toml")
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest ShedManifest
	if err := toml.Unmarshal(content, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &manifest, nil
}

// VerifySignature verifies the GPG signature of a .shed package
func (s *ShedInstaller) VerifySignature(shedFile, sigFile string) error {
	if !CommandExists("gpg") {
		return ErrGPGNotFound
	}

	// Verify the signature file exists
	if _, err := os.Stat(sigFile); os.IsNotExist(err) {
		return fmt.Errorf("signature file not found: %s", sigFile)
	}

	cmd := []string{"gpg", "--verify", sigFile, shedFile}
	return s.executor("", cmd)
}

// InstallFiles copies package files to the system with transaction support
func (s *ShedInstaller) InstallFiles(pkgDir, destRoot string, tx *Transaction) error {
	filesDir := filepath.Join(pkgDir, "files")

	// Check if files directory exists
	info, err := os.Stat(filesDir)
	if os.IsNotExist(err) {
		return nil // No files to install
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("files path is not a directory: %s", filesDir)
	}

	// Walk the files directory and install each file/dir
	return filepath.Walk(filesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path to determine destination
		relPath, err := filepath.Rel(filesDir, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		destPath := filepath.Join(destRoot, relPath)

		if info.IsDir() {
			// Track directory creation intent
			// Check if exists
			if _, err := os.Stat(destPath); os.IsNotExist(err) {
				tx.TrackDirectoryCreate(destPath)
				// Create directory
				cmd := []string{"sudo", "mkdir", "-p", destPath}
				if err := s.executor("", cmd); err != nil {
					return fmt.Errorf("failed to create directory %s: %w", destPath, err)
				}
			}
			return nil
		}

		// Handle file
		// 1. Track overwrite/create
		if err := tx.TrackOverwrite(destPath); err != nil {
			return fmt.Errorf("failed to track transaction for %s: %w", destPath, err)
		}

		// 2. Install file (copy)
		cmd := []string{"sudo", "cp", path, destPath}
		if err := s.executor("", cmd); err != nil {
			return fmt.Errorf("failed to install file %s: %w", destPath, err)
		}

		return nil
	})
}

// RunHooks executes package hooks (pre-install, post-install, etc.)
func (s *ShedInstaller) RunHooks(pkgDir, hookName string) error {
	hookPath := filepath.Join(pkgDir, "hooks", hookName+".sh")

	// Check if hook exists
	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		return nil // No hook to run
	}

	cmd := []string{"sh", hookPath}
	return s.executor("", cmd)
}

// Install performs a full .shed package installation
func (s *ShedInstaller) Install(shedFile string) error {
	// Start transaction
	tx, err := NewTransaction(s.executor)
	if err != nil {
		return fmt.Errorf("failed to initialize transaction: %w", err)
	}
	defer func() {
		if tx.active {
			output.Warning("Installation failed, rolling back changes...")
			if err := tx.Rollback(); err != nil {
				output.Error("Rollback failed: %v", err)
			}
		} else {
			// Clean up backups on success
			tx.Commit()
		}
	}()

	// Create extraction directory
	pkgName := filepath.Base(shedFile)
	extractDir := filepath.Join(s.cacheDir, "extracted", pkgName)

	// Extract package
	if err := s.Extract(shedFile, extractDir); err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}

	// Run pre-install hook
	if err := s.RunHooks(extractDir, "pre-install"); err != nil {
		return fmt.Errorf("pre-install hook failed: %w", err)
	}

	// Install files to system root with transaction
	if err := s.InstallFiles(extractDir, "/", tx); err != nil {
		return fmt.Errorf("file installation failed: %w", err)
	}

	// Run post-install hook
	if err := s.RunHooks(extractDir, "post-install"); err != nil {
		return fmt.Errorf("post-install hook failed: %w", err)
	}

	// Success - commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("transaction commit failed: %w", err)
	}

	return nil
}

// GetInstalledDir returns the directory where installed package manifests are stored
func (s *ShedInstaller) GetInstalledDir() string {
	return filepath.Join(s.cacheDir, "installed")
}

// IsInstalled checks if a package is installed
func (s *ShedInstaller) IsInstalled(pkgName string) bool {
	manifestPath := filepath.Join(s.GetInstalledDir(), pkgName, "manifest.toml")
	_, err := os.Stat(manifestPath)
	return err == nil
}

// Remove uninstalls a .shed package
func (s *ShedInstaller) Remove(pkgName string) error {
	installedDir := filepath.Join(s.GetInstalledDir(), pkgName)

	// Check if package is installed
	if !s.IsInstalled(pkgName) {
		return fmt.Errorf("package not installed: %s", pkgName)
	}

	// Read manifest to get file list
	manifest, err := s.ReadManifest(installedDir)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	// Run pre-remove hook
	if err := s.RunHooks(installedDir, "pre-remove"); err != nil {
		// Log warning but continue with removal
		fmt.Printf("Warning: pre-remove hook failed: %v\n", err)
	}

	// Remove installed files
	if err := s.RemoveFiles(manifest.Files); err != nil {
		return fmt.Errorf("failed to remove files: %w", err)
	}

	// Run post-remove hook
	if err := s.RunHooks(installedDir, "post-remove"); err != nil {
		// Log warning but continue
		fmt.Printf("Warning: post-remove hook failed: %v\n", err)
	}

	// Remove installed package directory
	if err := os.RemoveAll(installedDir); err != nil {
		return fmt.Errorf("failed to remove package directory: %w", err)
	}

	return nil
}

// RemoveFiles removes the specified files from the filesystem
func (s *ShedInstaller) RemoveFiles(files []string) error {
	for _, file := range files {
		// Use sudo to remove system files
		cmd := []string{"sudo", "rm", "-f", file}
		if err := s.executor("", cmd); err != nil {
			// Log warning but continue with other files
			fmt.Printf("Warning: failed to remove %s: %v\n", file, err)
		}
	}
	return nil
}

// ListInstalled returns a list of all installed .shed packages
func (s *ShedInstaller) ListInstalled() ([]string, error) {
	installedDir := s.GetInstalledDir()

	entries, err := os.ReadDir(installedDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var packages []string
	for _, entry := range entries {
		if entry.IsDir() {
			// Check if it has a manifest
			if s.IsInstalled(entry.Name()) {
				packages = append(packages, entry.Name())
			}
		}
	}

	return packages, nil
}

// InstalledPackageInfo holds info about an installed .shed package
type InstalledPackageInfo struct {
	Name        string
	Version     string
	Description string
}

// GetInstalledInfo returns detailed info about an installed .shed package
func (s *ShedInstaller) GetInstalledInfo(pkgName string) (*InstalledPackageInfo, error) {
	if !s.IsInstalled(pkgName) {
		return nil, fmt.Errorf("package not installed: %s", pkgName)
	}

	installedDir := filepath.Join(s.GetInstalledDir(), pkgName)
	manifest, err := s.ReadManifest(installedDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	return &InstalledPackageInfo{
		Name:        manifest.Name,
		Version:     manifest.Version,
		Description: manifest.Description,
	}, nil
}

// ListInstalledWithInfo returns all installed .shed packages with full info
func (s *ShedInstaller) ListInstalledWithInfo() ([]InstalledPackageInfo, error) {
	pkgNames, err := s.ListInstalled()
	if err != nil {
		return nil, err
	}

	var packages []InstalledPackageInfo
	for _, name := range pkgNames {
		info, err := s.GetInstalledInfo(name)
		if err != nil {
			// Skip packages with missing/invalid manifests
			continue
		}
		packages = append(packages, *info)
	}

	return packages, nil
}
