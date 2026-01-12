// Package backend provides error types for package manager operations.
package core

import (
	"errors"
	"fmt"
)

// Common backend errors (sentinels)
var (
	// ErrBackendNotFound is returned when no suitable backend is detected
	ErrBackendNotFound = errors.New("no suitable package manager backend found")

	// ErrPackageNotFound is returned when a package doesn't exist
	ErrPackageNotFound = errors.New("package not found")

	// ErrAURNotAvailable is returned when AUR is requested on non-Arch systems
	ErrAURNotAvailable = errors.New("AUR is only available on Arch-based distributions")

	// ErrNotInstalled is returned when trying to remove an uninstalled package
	ErrNotInstalled = errors.New("package is not installed")

	// ErrSyncFailed is returned when database sync fails
	ErrSyncFailed = errors.New("failed to sync package database")

	// ErrInstallFailed is returned when installation fails
	ErrInstallFailed = errors.New("package installation failed")

	// ErrRemoveFailed is returned when removal fails
	ErrRemoveFailed = errors.New("package removal failed")

	// ErrRootRequired is returned when operation requires root privileges
	ErrRootRequired = errors.New("operation requires root privileges")

	// ErrNetworkError is returned for connectivity issues
	ErrNetworkError = errors.New("network error")

	// ErrDependencyConflict is returned when packages have conflicting dependencies
	ErrDependencyConflict = errors.New("dependency conflict detected")

	// ErrBuildFailed is returned when building a package fails (e.g., AUR)
	ErrBuildFailed = errors.New("package build failed")

	// ErrInvalidPackage is returned when a package file is corrupt or invalid
	ErrInvalidPackage = errors.New("invalid package file")

	// ErrCancelled is returned when an operation is cancelled
	ErrCancelled = errors.New("operation cancelled")
)

// Operation represents the type of package manager operation
type Operation string

const (
	OpInstall  Operation = "install"
	OpRemove   Operation = "remove"
	OpUpgrade  Operation = "upgrade"
	OpSync     Operation = "sync"
	OpSearch   Operation = "search"
	OpInfo     Operation = "info"
	OpBuild    Operation = "build"
	OpDownload Operation = "download"
)

// PackageError is a structured error that includes package and operation context
type PackageError struct {
	Op       Operation // The operation that failed
	Package  string    // The package name (if applicable)
	Backend  string    // The backend name (pacman, etc.)
	Err      error     // The underlying error
	ExitCode int       // Exit code if from command execution
	Output   string    // Command output if available
}

// Error implements the error interface
func (e *PackageError) Error() string {
	if e.Package != "" {
		return fmt.Sprintf("%s %s (%s): %v", e.Op, e.Package, e.Backend, e.Err)
	}
	return fmt.Sprintf("%s (%s): %v", e.Op, e.Backend, e.Err)
}

// Unwrap returns the underlying error for errors.Is and errors.As
func (e *PackageError) Unwrap() error {
	return e.Err
}

// NewPackageError creates a new PackageError
func NewPackageError(op Operation, pkg, backend string, err error) *PackageError {
	return &PackageError{
		Op:      op,
		Package: pkg,
		Backend: backend,
		Err:     err,
	}
}

// WithExitCode adds an exit code to the error
func (e *PackageError) WithExitCode(code int) *PackageError {
	e.ExitCode = code
	return e
}

// WithOutput adds command output to the error
func (e *PackageError) WithOutput(output string) *PackageError {
	e.Output = output
	return e
}

// MultiPackageError represents an error affecting multiple packages
type MultiPackageError struct {
	Op       Operation
	Backend  string
	Packages []string
	Err      error
	Details  map[string]error // Per-package errors
}

// Error implements the error interface
func (e *MultiPackageError) Error() string {
	if len(e.Packages) == 1 {
		return fmt.Sprintf("%s %s (%s): %v", e.Op, e.Packages[0], e.Backend, e.Err)
	}
	return fmt.Sprintf("%s %d packages (%s): %v", e.Op, len(e.Packages), e.Backend, e.Err)
}

// Unwrap returns the underlying error
func (e *MultiPackageError) Unwrap() error {
	return e.Err
}

// NewMultiPackageError creates an error for operations on multiple packages
func NewMultiPackageError(op Operation, pkgs []string, backend string, err error) *MultiPackageError {
	return &MultiPackageError{
		Op:       op,
		Backend:  backend,
		Packages: pkgs,
		Err:      err,
		Details:  make(map[string]error),
	}
}

// AddPackageError adds a per-package error detail
func (e *MultiPackageError) AddPackageError(pkg string, err error) {
	e.Details[pkg] = err
}

// NetworkError provides context for network-related failures
type NetworkError struct {
	URL     string
	Op      string // "connect", "download", "upload"
	Err     error
	Timeout bool
}

// Error implements the error interface
func (e *NetworkError) Error() string {
	if e.Timeout {
		return fmt.Sprintf("network timeout during %s: %s", e.Op, e.URL)
	}
	return fmt.Sprintf("network error during %s (%s): %v", e.Op, e.URL, e.Err)
}

// Unwrap returns the underlying error
func (e *NetworkError) Unwrap() error {
	return e.Err
}

// Is allows matching against ErrNetworkError sentinel
func (e *NetworkError) Is(target error) bool {
	return target == ErrNetworkError
}

// NewNetworkError creates a new NetworkError
func NewNetworkError(op, url string, err error) *NetworkError {
	return &NetworkError{
		URL: url,
		Op:  op,
		Err: err,
	}
}

// WithTimeout marks the error as a timeout
func (e *NetworkError) WithTimeout() *NetworkError {
	e.Timeout = true
	return e
}

// PermissionError provides context for permission-related failures
type PermissionError struct {
	Op       Operation
	Resource string // File path or resource name
	Err      error
}

// Error implements the error interface
func (e *PermissionError) Error() string {
	if e.Resource != "" {
		return fmt.Sprintf("permission denied for %s on %s: %v", e.Op, e.Resource, e.Err)
	}
	return fmt.Sprintf("permission denied for %s: %v", e.Op, e.Err)
}

// Unwrap returns the underlying error
func (e *PermissionError) Unwrap() error {
	return e.Err
}

// Is allows matching against ErrRootRequired sentinel
func (e *PermissionError) Is(target error) bool {
	return target == ErrRootRequired
}

// NewPermissionError creates a new PermissionError
func NewPermissionError(op Operation, resource string, err error) *PermissionError {
	return &PermissionError{
		Op:       op,
		Resource: resource,
		Err:      err,
	}
}

// IsNotFound checks if an error indicates a package was not found
func IsNotFound(err error) bool {
	return errors.Is(err, ErrPackageNotFound)
}

// IsNetworkError checks if an error is network-related
func IsNetworkError(err error) bool {
	return errors.Is(err, ErrNetworkError)
}

// IsPermissionError checks if an error is permission-related
func IsPermissionError(err error) bool {
	return errors.Is(err, ErrRootRequired)
}

// IsAURError checks if an error is AUR-related
func IsAURError(err error) bool {
	return errors.Is(err, ErrAURNotAvailable)
}
