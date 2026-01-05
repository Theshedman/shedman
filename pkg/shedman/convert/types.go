package convert

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// File permission constants
const (
	DirPermissions  os.FileMode = 0755 // rwxr-xr-x for directories
	FilePermissions os.FileMode = 0644 // rw-r--r-- for regular files
	ExecPermissions os.FileMode = 0755 // rwxr-xr-x for executables
)

// Executor is a function type that executes command slices
type Executor func([]string) error

// DefaultExecutor is the default command executor
var DefaultExecutor Executor = func(cmd []string) error {
	if len(cmd) == 0 {
		return nil
	}
	c := exec.Command(cmd[0], cmd[1:]...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// CommandExists checks if a command is available in PATH
func CommandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// ShedManifest represents the manifest.toml in a .shed package
type ShedManifest struct {
	// Required fields
	Name        string `toml:"name"`
	Version     string `toml:"version"`
	Description string `toml:"description"`

	// Architecture and platform
	Arch string `toml:"arch"` // e.g., "x86_64", "any"

	// Package metadata
	Url       string `toml:"url"`
	License   string `toml:"license"`
	Packager  string `toml:"packager"`
	BuildDate string `toml:"build_date"` // ISO 8601 format

	// Dependencies and relationships
	Depends     []string `toml:"depends"`
	OptDepends  []string `toml:"optdepends"`
	MakeDepends []string `toml:"makedepends"`
	Provides    []string `toml:"provides"`
	Conflicts   []string `toml:"conflicts"`
	Replaces    []string `toml:"replaces"`

	// Size information
	Size          int64 `toml:"size"`           // Package size in bytes
	InstalledSize int64 `toml:"installed_size"` // Installed size in bytes

	// Backup files (preserved on upgrade)
	Backup []string `toml:"backup"`

	// Package groups
	Groups []string `toml:"groups"`
}

// InstallShed installs a .shed package using tar extraction
func InstallShed(shedPath string) error {
	if !CommandExists("tar") {
		return ErrTarNotFound
	}

	// Create temp extraction directory
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	extractDir := filepath.Join(home, ".cache", "shedman", "shed", "extracted", filepath.Base(shedPath))
	if err := os.MkdirAll(extractDir, DirPermissions); err != nil {
		return fmt.Errorf("failed to create extraction directory: %w", err)
	}

	// Extract package
	if err := DefaultExecutor([]string{"tar", "-xf", shedPath, "-C", extractDir}); err != nil {
		return fmt.Errorf("failed to extract shed package: %w", err)
	}

	// Run pre-install hook if exists
	preInstall := filepath.Join(extractDir, "hooks", "pre-install.sh")
	if _, err := os.Stat(preInstall); err == nil {
		if err := DefaultExecutor([]string{"sh", preInstall}); err != nil {
			fmt.Printf("Warning: pre-install hook failed: %v\n", err)
			// Continue installation despite hook failure
		}
	}

	// Install files to root
	filesDir := filepath.Join(extractDir, "files")
	if _, err := os.Stat(filesDir); err == nil {
		if err := DefaultExecutor([]string{"sudo", "cp", "-r", filesDir + "/.", "/"}); err != nil {
			return fmt.Errorf("failed to install files: %w", err)
		}
	}

	// Run post-install hook if exists
	postInstall := filepath.Join(extractDir, "hooks", "post-install.sh")
	if _, err := os.Stat(postInstall); err == nil {
		if err := DefaultExecutor([]string{"sh", postInstall}); err != nil {
			fmt.Printf("Warning: post-install hook failed: %v\n", err)
			// Warning only, installation is complete
		}
	}

	return nil
}

// ErrTarNotFound is returned when tar is not found
var ErrTarNotFound = fmt.Errorf("tar is required for extraction but not found in PATH")
