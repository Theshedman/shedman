package core

import (
	"errors"
	"testing"

)

func TestPackageError(t *testing.T) {
	err := NewPackageError(
		OpInstall,
		"vim",
		"pacman",
		ErrInstallFailed,
	)

	if err.Op != OpInstall {
		t.Errorf("Expected OpInstall, got %s", err.Op)
	}
	if err.Package != "vim" {
		t.Errorf("Expected package 'vim', got '%s'", err.Package)
	}
	if err.Backend != "pacman" {
		t.Errorf("Expected backend 'pacman', got '%s'", err.Backend)
	}

	// Test Error() output
	errStr := err.Error()
	if errStr != "install vim (pacman): package installation failed" {
		t.Errorf("Unexpected error string: %s", errStr)
	}

	// Test Unwrap
	if !errors.Is(err, ErrInstallFailed) {
		t.Error("Expected error to wrap ErrInstallFailed")
	}
}

func TestPackageError_WithExitCode(t *testing.T) {
	err := NewPackageError(
		OpInstall,
		"git",
		"pacman",
		ErrInstallFailed,
	).WithExitCode(1)

	if err.ExitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", err.ExitCode)
	}
}

func TestPackageError_WithOutput(t *testing.T) {
	err := NewPackageError(
		OpInstall,
		"git",
		"pacman",
		ErrInstallFailed,
	).WithOutput("error: target not found: git")

	if err.Output != "error: target not found: git" {
		t.Errorf("Expected output, got '%s'", err.Output)
	}
}

func TestPackageError_NoPackage(t *testing.T) {
	err := NewPackageError(
		OpSync,
		"",
		"pacman",
		ErrSyncFailed,
	)

	errStr := err.Error()
	if errStr != "sync (pacman): failed to sync package database" {
		t.Errorf("Unexpected error string: %s", errStr)
	}
}

func TestMultiPackageError(t *testing.T) {
	err := NewMultiPackageError(
		OpInstall,
		[]string{"vim", "git", "htop"},
		"pacman",
		ErrInstallFailed,
	)

	if len(err.Packages) != 3 {
		t.Errorf("Expected 3 packages, got %d", len(err.Packages))
	}

	errStr := err.Error()
	if errStr != "install 3 packages (pacman): package installation failed" {
		t.Errorf("Unexpected error string: %s", errStr)
	}

	// Test AddPackageError
	err.AddPackageError("vim", errors.New("conflict with neovim"))
	if err.Details["vim"] == nil {
		t.Error("Expected per-package error for vim")
	}
}

func TestMultiPackageError_SinglePackage(t *testing.T) {
	err := NewMultiPackageError(
		OpInstall,
		[]string{"vim"},
		"pacman",
		ErrInstallFailed,
	)

	errStr := err.Error()
	if errStr != "install vim (pacman): package installation failed" {
		t.Errorf("Unexpected error string: %s", errStr)
	}
}

func TestNetworkError(t *testing.T) {
	err := NewNetworkError(
		"download",
		"https://aur.archlinux.org/packages/vim",
		errors.New("connection refused"),
	)

	if err.URL != "https://aur.archlinux.org/packages/vim" {
		t.Errorf("Expected URL, got '%s'", err.URL)
	}

	// Test Is() for sentinel matching
	if !errors.Is(err, ErrNetworkError) {
		t.Error("Expected error to match ErrNetworkError sentinel")
	}
}

func TestNetworkError_Timeout(t *testing.T) {
	err := NewNetworkError(
		"connect",
		"https://aur.archlinux.org",
		nil,
	).WithTimeout()

	if !err.Timeout {
		t.Error("Expected timeout flag")
	}

	errStr := err.Error()
	if errStr != "network timeout during connect: https://aur.archlinux.org" {
		t.Errorf("Unexpected error string: %s", errStr)
	}
}

func TestPermissionError(t *testing.T) {
	err := NewPermissionError(
		OpInstall,
		"/var/lib/pacman",
		errors.New("permission denied"),
	)

	if err.Resource != "/var/lib/pacman" {
		t.Errorf("Expected resource, got '%s'", err.Resource)
	}

	// Test Is() for sentinel matching
	if !errors.Is(err, ErrRootRequired) {
		t.Error("Expected error to match ErrRootRequired sentinel")
	}
}

func TestHelperFunctions(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		checkFn  func(error) bool
		expected bool
	}{
		{"IsNotFound true", ErrPackageNotFound, IsNotFound, true},
		{"IsNotFound false", ErrInstallFailed, IsNotFound, false},
		{"IsNetworkError true", NewNetworkError("test", "", nil), IsNetworkError, true},
		{"IsNetworkError false", ErrInstallFailed, IsNetworkError, false},
		{"IsPermissionError true", NewPermissionError(OpInstall, "", nil), IsPermissionError, true},
		{"IsPermissionError false", ErrInstallFailed, IsPermissionError, false},
		{"IsAURError true", ErrAURNotAvailable, IsAURError, true},
		{"IsAURError false", ErrInstallFailed, IsAURError, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.checkFn(tc.err) != tc.expected {
				t.Errorf("Expected %v for %s", tc.expected, tc.name)
			}
		})
	}
}

func TestWrappedPackageError(t *testing.T) {
	// Test that wrapped errors still match sentinels
	pkgErr := NewPackageError(
		OpInstall,
		"vim",
		"pacman",
		ErrPackageNotFound,
	)

	if !IsNotFound(pkgErr) {
		t.Error("Wrapped PackageError should match IsNotFound via Unwrap")
	}
}
