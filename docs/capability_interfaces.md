# Capability-Based Interface Specification

## Overview

shedman uses **capability-based interface design** where backends declare their actual capabilities through interface composition, rather than forcing all backends to implement all features.

This specification defines the core interfaces and optional capabilities for the shedman package manager backend system.

---

## Design Principles

1. **Interface Segregation**: Many small, focused interfaces instead of one large interface
2. **Optional Capabilities**: Backends implement only what they support
3. **Runtime Discovery**: Check capabilities at runtime via type assertions
4. **Graceful Degradation**: Fallback when capabilities unavailable
5. **Compile-Time Safety**: Type system enforces correct usage

---

## Core Interface

### Backend

Every backend MUST implement this minimal interface:

```go
package backend

// Backend is the minimal interface all backends must implement
type Backend interface {
    // Name returns the backend identifier (e.g., "pacman", "apt")
    Name() string
    
    // Sync refreshes package databases
    Sync() error
    
    // IsAvailable checks if this backend is available on the system
    IsAvailable() bool
}
```

**Rationale:**

- All backends can sync their databases
- All backends have a name
- All backends must self-report availability

---

## Capability Interfaces

### 1. PackageManager

Basic package installation and removal:

```go
// PackageManager provides core package management operations
type PackageManager interface {
    Backend  // Embeds Backend interface
    
    // Install installs packages from repositories
    Install(pkgs []string, opts InstallOptions) error
    
    // Remove removes installed packages
    Remove(pkgs []string, opts RemoveOptions) error
    
    // IsInstalled checks if a package is currently installed
    IsInstalled(pkg string) bool
}

// InstallOptions configures installation behavior
type InstallOptions struct {
    NoConfirm     bool     // Skip confirmation prompts
    AsD eps        bool     // Mark as dependency
    AsExplicit    bool     // Mark as explicitly installed
    Needed        bool     // Skip if already installed
    DownloadOnly  bool     // Download without installing
    Overwrite     []string // Glob patterns for files to overwrite
}

// RemoveOptions configures removal behavior
type RemoveOptions struct {
    NoConfirm   bool  // Skip confirmation prompts
    Recursive   bool  // Remove unused dependencies
    NoSave      bool  // Don't create .pacsave files
    Unneeded    bool  // Remove unneeded packages
}
```

**Usage:**

```go
backend := GetBackend()

// Check if backend supports package management
pm, ok := backend.(PackageManager)
if !ok {
    return errors.New("backend doesn't support package management")
}

// Use the capability
err := pm.Install([]string{"neovim"}, InstallOptions{
    NoConfirm: true,
    Needed: true,
})
```

---

### 2. Searchable

Package searching capability:

```go
// Searchable backends can search for packages
type Searchable interface {
    // Search searches for packages by name or description
    // Returns list of matching packages
    Search(query string) ([]PackageInfo, error)
}

// PackageInfo describes a package
type PackageInfo struct {
    Name         string
    Version      string
    Description  string
    Repository   string   // Source repository
    Installed    bool     // Is currently installed
    Size         int64    // Package size in bytes
    Dependencies []string // Package dependencies
    Groups       []string // Package groups
    Licenses     []string // License identifiers
    URL          string   // Project URL
}
```

**Usage:**

```go
if searcher, ok := backend.(Searchable); ok {
    results, err := searcher.Search("vim")
    if err != nil {
        return err
    }
    
    for _, pkg := range results {
        fmt.Printf("%s %s - %s\n", pkg.Name, pkg.Version, pkg.Description)
    }
} else {
    return errors.New("search not supported by this backend")
}
```

---

### 3. Informer

Detailed package information:

```go
// Informer backends provide detailed package information
type Informer interface {
    // Info retrieves detailed information about a package
    Info(pkgName string) (*PackageInfo, error)
    
    // GetInstalledPackages returns all installed packages
    GetInstalledPackages() ([]PackageInfo, error)
}
```

**Usage:**

```go
if informer, ok := backend.(Informer); ok {
    info, err := informer.Info("neovim")
    if err != nil {
        return err
    }
    
    fmt.Printf("Name: %s\n", info.Name)
    fmt.Printf("Version: %s\n", info.Version)
    fmt.Printf("Dependencies: %s\n", strings.Join(info.Dependencies, ", "))
}
```

---

### 4. Upgradable

System and package upgrades:

```go
// Upgradable backends support package upgrades
type Upgradable interface {
    // Upgrade upgrades packages or entire system
    // Empty pkgs slice = upgrade all
    Upgrade(pkgs []string, opts UpgradeOptions) error
}

// UpgradeOptions configures upgrade behavior
type UpgradeOptions struct {
    NoConfirm      bool     // Skip confirmation prompts
    DownloadOnly   bool     // Download without installing
    Ignore         []string // Packages to skip
    TargetBackends []string // Backend filter (for multi-backend)
}
```

**Usage:**

```go
if upgrader, ok := backend.(Upgradable); ok {
    // Full system upgrade
    err := upgrader.Upgrade(nil, UpgradeOptions{
        NoConfirm: false,  // Prompt user
        Ignore: []string{"linux"},  // Don't upgrade kernel
    })
}
```

---

### 5. LocalInstaller

Installing local package files:

```go
// LocalInstaller backends can install local package files
type LocalInstaller interface {
    // InstallLocal installs from a local file (.pkg.tar.zst, .deb, .rpm)
    InstallLocal(path string, opts InstallOptions) error
}
```

**Usage:**

```go
if installer, ok := backend.(LocalInstaller); ok {
    err := installer.InstallLocal("/path/to/package.pkg.tar.zst", InstallOptions{
        NoConfirm: true,
    })
}
```

---

### 6. FileProvider

Query package file ownership:

```go
// FileProvider backends can query file ownership and package contents
type FileProvider interface {
    // GetPackageFiles returns list of files owned by package
    GetPackageFiles(pkgName string) ([]string, error)
    
    // GetFileOwner returns the package that owns a file
    GetFileOwner(path string) (string, error)
}
```

**Usage:**

```go
if fp, ok := backend.(FileProvider); ok {
    files, err := fp.GetPackageFiles("neovim")
    // ["/usr/bin/nvim", "/usr/share/nvim/...", ...]
    
    owner, err := fp.GetFileOwner("/usr/bin/nvim")
    // "neovim"
}
```

---

### 7. GroupManager

Package group operations:

```go
// GroupManager backends support package groups
type GroupManager interface {
    // ListGroups returns all available groups
    ListGroups() ([]Group, error)
    
    // GetGroup retrieves group details
    GetGroup(name string) (*Group, error)
    
    // InstallGroup installs all packages in a group
    InstallGroup(name string, opts InstallOptions) error
}

// Group represents a package group
type Group struct {
    Name        string
    Description string
    Packages    []string
}
```

**Usage:**

```go
if gm, ok := backend.(GroupManager); ok {
    groups, err := gm.ListGroups()
    
    // Install development tools group
    err = gm.InstallGroup("base-devel", InstallOptions{
        NoConfirm: true,
    })
}
```

---

## Complete Backend Example

### Pacman Backend Implementation

```go
package pacman

import (
    "github.com/theshedman/shedman/pkg/backend"
)

// Backend implements multiple capability interfaces for pacman
type Backend struct {
    binaryPath string
    executor   CommandExecutor
}

// Verify Backend implements all interfaces at compile time
var (
    _ backend.Backend        = (*Backend)(nil)
    _ backend.PackageManager = (*Backend)(nil)
    _ backend.Searchable     = (*Backend)(nil)
    _ backend.Informer       = (*Backend)(nil)
    _ backend.Upgradable     = (*Backend)(nil)
    _ backend.LocalInstaller = (*Backend)(nil)
    _ backend.FileProvider   = (*Backend)(nil)
    _ backend.GroupManager   = (*Backend)(nil)
)

// ===== Core Backend Interface =====

func (b *Backend) Name() string {
    return "pacman"
}

func (b *Backend) Sync() error {
    return b.executor.Run("pacman", "-Sy")
}

func (b *Backend) IsAvailable() bool {
    _, err := exec.LookPath(b.binaryPath)
    return err == nil
}

// ===== PackageManager Interface =====

func (b *Backend) Install(pkgs []string, opts backend.InstallOptions) error {
    args := []string{"-S"}
    
    if opts.NoConfirm {
        args = append(args, "--noconfirm")
    }
    if opts.AsDeps {
        args = append(args, "--asdeps")
    }
    if opts.Needed {
        args = append(args, "--needed")
    }
    
    args = append(args, pkgs...)
    return b.executor.Run("sudo", append([]string{b.binaryPath}, args...)...)
}

func (b *Backend) Remove(pkgs []string, opts backend.RemoveOptions) error {
    args := []string{"-R"}
    
    if opts.NoConfirm {
        args = append(args, "--noconfirm")
    }
    if opts.Recursive {
        args = append(args, "--recursive")
    }
    if opts.NoSave {
        args = append(args, "--nosave")
    }
    
    args = append(args, pkgs...)
    return b.executor.Run("sudo", append([]string{b.binaryPath}, args...)...)
}

func (b *Backend) IsInstalled(pkg string) bool {
    err := b.executor.Run(b.binaryPath, "-Qi", pkg)
    return err == nil
}

// ===== Searchable Interface =====

func (b *Backend) Search(query string) ([]backend.PackageInfo, error) {
    output, err := b.executor.Output(b.binaryPath, "-Ss", query)
    if err != nil {
        return nil, err
    }
    
    return parsePacmanSearchOutput(string(output)), nil
}

// ===== Informer Interface =====

func (b *Backend) Info(pkgName string) (*backend.PackageInfo, error) {
    // Try installed packages first
    output, err := b.executor.Output(b.binaryPath, "-Qi", pkgName)
    if err != nil {
        // Try sync databases
        output, err = b.executor.Output(b.binaryPath, "-Si", pkgName)
        if err != nil {
            return nil, backend.ErrPackageNotFound
        }
    }
    
    return parsePacmanInfo(string(output)), nil
}

func (b *Backend) GetInstalledPackages() ([]backend.PackageInfo, error) {
    output, err := b.executor.Output(b.binaryPath, "-Q")
    if err != nil {
        return nil, err
    }
    
    return parsePacmanList(string(output)), nil
}

// ===== Upgradable Interface =====

func (b *Backend) Upgrade(pkgs []string, opts backend.UpgradeOptions) error {
    args := []string{"-Su"}
    
    if opts.NoConfirm {
        args = append(args, "--noconfirm")
    }
    
    for _, ignore := range opts.Ignore {
        args = append(args, "--ignore", ignore)
    }
    
    if len(pkgs) > 0 {
        args = append(args, pkgs...)
    }
    
    return b.executor.Run("sudo", append([]string{b.binaryPath}, args...)...)
}

// ===== LocalInstaller Interface =====

func (b *Backend) InstallLocal(path string, opts backend.InstallOptions) error {
    args := []string{"-U"}
    
    if opts.NoConfirm {
        args = append(args, "--noconfirm")
    }
    
    args = append(args, path)
    return b.executor.Run("sudo", append([]string{b.binaryPath}, args...)...)
}

// ===== FileProvider Interface =====

func (b *Backend) GetPackageFiles(pkgName string) ([]string, error) {
    output, err := b.executor.Output(b.binaryPath, "-Ql", pkgName)
    if err != nil {
        return nil, err
    }
    
    return parseFileList(string(output)), nil
}

func (b *Backend) GetFileOwner(path string) (string, error) {
    output, err := b.executor.Output(b.binaryPath, "-Qo", path)
    if err != nil {
        return "", err
    }
    
    return parseOwner(string(output)), nil
}

// ===== GroupManager Interface =====

func (b *Backend) ListGroups() ([]backend.Group, error) {
    output, err := b.executor.Output(b.binaryPath, "-Sg")
    if err != nil {
        return nil, err
    }
    
    return parseGroups(string(output)), nil
}

func (b *Backend) GetGroup(name string) (*backend.Group, error) {
    output, err := b.executor.Output(b.binaryPath, "-Sg", name)
    if err != nil {
        return nil, err
    }
    
    return parseGroupDetail(string(output)), nil
}

func (b *Backend) InstallGroup(name string, opts backend.InstallOptions) error {
    // Groups are installed like packages in pacman
    return b.Install([]string{name}, opts)
}
```

---

## Usage Patterns

### Pattern 1: Required Capability

```go
// Function requires specific capability
func InstallPackages(backend backend.Backend, pkgs []string) error {
    pm, ok := backend.(backend.PackageManager)
    if !ok {
        return ErrCapabilityRequired("PackageManager")
    }
    
    return pm.Install(pkgs, backend.InstallOptions{})
}
```

### Pattern 2: Optional Capability with Fallback

```go
// Function uses capability if available, falls back otherwise
func SearchPackages(backend backend.Backend, query string) ([]backend.PackageInfo, error) {
    // Prefer search capability
    if searcher, ok := backend.(backend.Searchable); ok {
        return searcher.Search(query)
    }
    
    // Fallback: list all then filter manually
    if informer, ok := backend.(backend.Informer); ok {
        all, err := informer.GetInstalledPackages()
        if err != nil {
            return nil, err
        }
        
        return filterPackages(all, query), nil
    }
    
    return nil, ErrSearchNotSupported
}
```

### Pattern 3: Capability Detection Helper

```go
// Helper function to check multiple capabilities
func GetBackendCapabilities(backend backend.Backend) []string {
    caps := []string{}
    
    if _, ok := backend.(backend.PackageManager); ok {
        caps = append(caps, "PackageManager")
    }
    if _, ok := backend.(backend.Searchable); ok {
        caps = append(caps, "Searchable")
    }
    if _, ok := backend.(backend.Informer); ok {
        caps = append(caps, "Informer")
    }
    if _, ok := backend.(backend.Upgradable); ok {
        caps = append(caps, "Upgradable")
    }
    if _, ok := backend.(backend.LocalInstaller); ok {
        caps = append(caps, "LocalInstaller")
    }
    if _, ok := backend.(backend.FileProvider); ok {
        caps = append(caps, "FileProvider")
    }
    if _, ok := backend.(backend.GroupManager); ok {
        caps = append(caps, "GroupManager")
    }
    
    return caps
}
```

### Pattern 4: Engine with Capability Routing

```go
// Engine delegates to backend capabilities
type Engine struct {
    backend backend.Backend
}

func (e *Engine) Install(pkgs []string, opts InstallOptions) error {
    pm, ok := e.backend.(backend.PackageManager)
    if !ok {
        return ErrNoBackend("install")
    }
    
    return pm.Install(pkgs, opts)
}

func (e *Engine) Search(query string) ([]PackageInfo, error) {
    searcher, ok := e.backend.(backend.Searchable)
    if !ok {
        return nil, ErrNoBackend("search")
    }
    
    return searcher.Search(query)
}
```

---

## Error Handling

### Standard Errors

```go
package backend

var (
    // ErrBackendNotFound indicates no backend is available
    ErrBackendNotFound = errors.New("no backend found")
    
    // ErrPackageNotFound indicates package doesn't exist
    ErrPackageNotFound = errors.New("package not found")
    
    // ErrPackageExists indicates package already installed
    ErrPackageExists = errors.New("package already installed")
    
    // ErrCapabilityNotSupported indicates missing capability
    ErrCapabilityNotSupported = errors.New("capability not supported")
)

// ErrNoBackend creates capability error
func ErrNoBackend(operation string) error {
    return fmt.Errorf("no backend supports operation: %s", operation)
}
```

### Capability-Specific Errors

```go
// Check capability before use
pm, ok := backend.(backend.PackageManager)
if !ok {
    return backend.ErrCapabilityNotSupported
}

// Try operation, handle errors
err := pm.Install(pkgs, opts)
if err != nil {
    if errors.Is(err, backend.ErrPackageNotFound) {
        // Handle not found
    } else if errors.Is(err, backend.ErrPackageExists) {
        // Already installed
    } else {
        // Other error
        return err
    }
}
```

---

## Testing

### Mock Backend

```go
// MockBackend for testing
type MockBackend struct {
    mock.Mock
}

// Implement only needed capabilities
func (m *MockBackend) Name() string {
    args := m.Called()
    return args.String(0)
}

func (m *MockBackend) Sync() error {
    args := m.Called()
    return args.Error(0)
}

func (m *MockBackend) IsAvailable() bool {
    args := m.Called()
    return args.Bool(0)
}

// PackageManager capability
func (m *MockBackend) Install(pkgs []string, opts InstallOptions) error {
    args := m.Called(pkgs, opts)
    return args.Error(0)
}

func (m *MockBackend) Remove(pkgs []string, opts RemoveOptions) error {
    args := m.Called(pkgs, opts)
    return args.Error(0)
}

func (m *MockBackend) IsInstalled(pkg string) bool {
    args := m.Called(pkg)
    return args.Bool(0)
}

// Searchable capability
func (m *MockBackend) Search(query string) ([]PackageInfo, error) {
    args := m.Called(query)
    return args.Get(0).([]PackageInfo), args.Error(1)
}
```

### Test Example

```go
func TestEngineInstall(t *testing.T) {
    mock := new(MockBackend)
    mock.On("Install", []string{"neovim"}, InstallOptions{}).Return(nil)
    
    engine := NewEngine(mock)
    err := engine.Install([]string{"neovim"}, InstallOptions{})
    
    assert.NoError(t, err)
    mock.AssertExpectations(t)
}

func TestEngineSearchWithoutCapability(t *testing.T) {
    // Mock without Searchable interface
    mock := new(MockBackend)
    
    engine := NewEngine(mock)
    _, err := engine.Search("vim")
    
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "search")
}
```

---

## Summary

**Capability Interfaces Defined:**

| Interface | Purpose | Methods |
|-----------|---------|---------|
| `Backend` | Core (required) | Name, Sync, IsAvailable |
| `PackageManager` | Install/remove | Install, Remove, IsInstalled |
| `Searchable` | Search | Search |
| `Informer` | Info queries | Info, GetInstalledPackages |
| `Upgradable` | Upgrades | Upgrade |
| `LocalInstaller` | Local files | InstallLocal |
| `FileProvider` | File queries | GetPackageFiles, GetFileOwner |
| `GroupManager` | Package groups | ListGroups, GetGroup, InstallGroup |

**Benefits:**

✅ **Flexible**: Backends implement what they support  
✅ **Type-Safe**: Compile-time interface verification  
✅ **Testable**: Easy to mock individual capabilities  
✅ **Degradable**: Graceful handling of missing capabilities  
✅ **Extensible**: New capabilities added without breaking existing

**pacman Backend**: Implements ALL 8 interfaces (full-featured)

**Future Backends** (if added): Can implement subset based on their actual capabilities
