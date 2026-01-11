package backend_test

import (
	"errors"
	"testing"

	"github.com/theshedman/shedman/pkg/shedman/backend"
)

func TestPackageError(t *testing.T) {
	err := backend.NewPackageError(
		backend.OpInstall,
		"vim",
		"pacman",
		backend.ErrInstallFailed,
	)

	if err.Op != backend.OpInstall {
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
	if !errors.Is(err, backend.ErrInstallFailed) {
		t.Error("Expected error to wrap ErrInstallFailed")
	}
}

func TestPackageError_WithExitCode(t *testing.T) {
	err := backend.NewPackageError(
		backend.OpInstall,
		"git",
		"pacman",
		backend.ErrInstallFailed,
	).WithExitCode(1)

	if err.ExitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", err.ExitCode)
	}
}

func TestPackageError_WithOutput(t *testing.T) {
	err := backend.NewPackageError(
		backend.OpInstall,
		"git",
		"pacman",
		backend.ErrInstallFailed,
	).WithOutput("error: target not found: git")

	if err.Output != "error: target not found: git" {
		t.Errorf("Expected output, got '%s'", err.Output)
	}
}

func TestPackageError_NoPackage(t *testing.T) {
	err := backend.NewPackageError(
		backend.OpSync,
		"",
		"pacman",
		backend.ErrSyncFailed,
	)

	errStr := err.Error()
	if errStr != "sync (pacman): failed to sync package database" {
		t.Errorf("Unexpected error string: %s", errStr)
	}
}

func TestMultiPackageError(t *testing.T) {
	err := backend.NewMultiPackageError(
		backend.OpInstall,
		[]string{"vim", "git", "htop"},
		"pacman",
		backend.ErrInstallFailed,
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
	err := backend.NewMultiPackageError(
		backend.OpInstall,
		[]string{"vim"},
		"pacman",
		backend.ErrInstallFailed,
	)

	errStr := err.Error()
	if errStr != "install vim (pacman): package installation failed" {
		t.Errorf("Unexpected error string: %s", errStr)
	}
}

func TestNetworkError(t *testing.T) {
	err := backend.NewNetworkError(
		"download",
		"https://aur.archlinux.org/packages/vim",
		errors.New("connection refused"),
	)

	if err.URL != "https://aur.archlinux.org/packages/vim" {
		t.Errorf("Expected URL, got '%s'", err.URL)
	}

	// Test Is() for sentinel matching
	if !errors.Is(err, backend.ErrNetworkError) {
		t.Error("Expected error to match ErrNetworkError sentinel")
	}
}

func TestNetworkError_Timeout(t *testing.T) {
	err := backend.NewNetworkError(
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
	err := backend.NewPermissionError(
		backend.OpInstall,
		"/var/lib/pacman",
		errors.New("permission denied"),
	)

	if err.Resource != "/var/lib/pacman" {
		t.Errorf("Expected resource, got '%s'", err.Resource)
	}

	// Test Is() for sentinel matching
	if !errors.Is(err, backend.ErrRootRequired) {
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
		{"IsNotFound true", backend.ErrPackageNotFound, backend.IsNotFound, true},
		{"IsNotFound false", backend.ErrInstallFailed, backend.IsNotFound, false},
		{"IsNetworkError true", backend.NewNetworkError("test", "", nil), backend.IsNetworkError, true},
		{"IsNetworkError false", backend.ErrInstallFailed, backend.IsNetworkError, false},
		{"IsPermissionError true", backend.NewPermissionError(backend.OpInstall, "", nil), backend.IsPermissionError, true},
		{"IsPermissionError false", backend.ErrInstallFailed, backend.IsPermissionError, false},
		{"IsAURError true", backend.ErrAURNotAvailable, backend.IsAURError, true},
		{"IsAURError false", backend.ErrInstallFailed, backend.IsAURError, false},
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
	pkgErr := backend.NewPackageError(
		backend.OpInstall,
		"vim",
		"pacman",
		backend.ErrPackageNotFound,
	)

	if !backend.IsNotFound(pkgErr) {
		t.Error("Wrapped PackageError should match IsNotFound via Unwrap")
	}
}
