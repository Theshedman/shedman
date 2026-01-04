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
	Name        string   `toml:"name"`
	Version     string   `toml:"version"`
	Description string   `toml:"description"`
	Depends     []string `toml:"depends"`
	Provides    []string `toml:"provides"`
	Conflicts   []string `toml:"conflicts"`
}

// InstallShed installs a .shed package using tar extraction
func InstallShed(shedPath string) error {
	if !CommandExists("tar") {
		return ErrTarNotFound
	}

	// Create temp extraction directory
	home, _ := os.UserHomeDir()
	extractDir := home + "/.cache/shedman/shed/extracted/" + filepath.Base(shedPath)
	os.MkdirAll(extractDir, DirPermissions)

	// Extract package
	err := DefaultExecutor([]string{"tar", "-xf", shedPath, "-C", extractDir})
	if err != nil {
		return fmt.Errorf("failed to extract shed package: %w", err)
	}

	// Run pre-install hook if exists
	preInstall := extractDir + "/hooks/pre-install.sh"
	if _, err := os.Stat(preInstall); err == nil {
		DefaultExecutor([]string{"sh", preInstall})
	}

	// Install files to root
	filesDir := extractDir + "/files"
	if _, err := os.Stat(filesDir); err == nil {
		err = DefaultExecutor([]string{"sudo", "cp", "-r", filesDir + "/.", "/"})
		if err != nil {
			return fmt.Errorf("failed to install files: %w", err)
		}
	}

	// Run post-install hook if exists
	postInstall := extractDir + "/hooks/post-install.sh"
	if _, err := os.Stat(postInstall); err == nil {
		DefaultExecutor([]string{"sh", postInstall})
	}

	return nil
}

// ErrTarNotFound is returned when tar is not found
var ErrTarNotFound = fmt.Errorf("tar is required for extraction but not found in PATH")
