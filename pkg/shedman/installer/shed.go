package installer

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
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
}

// ShedInstaller handles .shed package installation
type ShedInstaller struct {
	executor Executor
	cacheDir string
}

// NewShedInstaller creates a new ShedInstaller
func NewShedInstaller() *ShedInstaller {
	home, _ := os.UserHomeDir()
	return &ShedInstaller{
		executor: DefaultExecutor,
		cacheDir: filepath.Join(home, ".cache", "shedman", "shed"),
	}
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
	return s.executor(cmd)
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
	return s.executor(cmd)
}

// InstallFiles copies package files to the system
func (s *ShedInstaller) InstallFiles(pkgDir, destRoot string) error {
	filesDir := filepath.Join(pkgDir, "files")

	// Check if files directory exists
	if _, err := os.Stat(filesDir); os.IsNotExist(err) {
		return nil // No files to install
	}

	// Use rsync or cp to install files
	cmd := []string{"sudo", "cp", "-r", filesDir + "/.", destRoot}
	return s.executor(cmd)
}

// RunHooks executes package hooks (pre-install, post-install, etc.)
func (s *ShedInstaller) RunHooks(pkgDir, hookName string) error {
	hookPath := filepath.Join(pkgDir, "hooks", hookName+".sh")

	// Check if hook exists
	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		return nil // No hook to run
	}

	cmd := []string{"sh", hookPath}
	return s.executor(cmd)
}

// Install performs a full .shed package installation
func (s *ShedInstaller) Install(shedFile string) error {
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

	// Install files to system root
	if err := s.InstallFiles(extractDir, "/"); err != nil {
		return fmt.Errorf("file installation failed: %w", err)
	}

	// Run post-install hook
	if err := s.RunHooks(extractDir, "post-install"); err != nil {
		return fmt.Errorf("post-install hook failed: %w", err)
	}

	return nil
}
