package cmd

import (
	"github.com/theshedman/shedman/pkg/core"
)

// MockBackend implements core.OfficialBackend and core.Maintainer for testing
type MockBackend struct {
	NameFunc          func() string
	IsAvailableFunc   func() bool
	SyncFunc          func() error
	InstallFunc       func(pkgs []string, opts core.InstallOptions) error
	RemoveFunc        func(pkgs []string, opts core.RemoveOptions) error
	UpgradeFunc       func(pkgs []string, opts core.UpgradeOptions) error
	IsInstalledFunc   func(pkgName string) bool
	InfoFunc          func(pkgName string) (*core.PackageInfo, error)
	PkgFilesFunc      func(pkgName string) ([]string, error)
	InstalledPkgsFunc func() ([]core.PackageInfo, error)
	InstallLocalFunc  func(path string, opts core.InstallOptions) error
	SearchFunc        func(query string) ([]core.PackageInfo, error)

	// Maintainer
	CleanCacheFunc    func(opts core.CleanOptions) error
	ListOrphansFunc   func() ([]string, error)
	RemoveOrphansFunc func(pkgs []string) error

	// FileProvider
	GetFileOwnerFunc func(path string) (string, error)
	SearchFilesFunc  func(query string) ([]string, error)

	// Verifier
	VerifyAllFunc     func() (map[string][]string, error)
	VerifyPackageFunc func(pkgName string) ([]string, error)

	// Builder
	BuildFunc func(dir string, opts core.BuildOptions) error

	// KeyManager
	InitKeyringFunc func() error
	RefreshKeysFunc func() error
	ListKeysFunc    func() ([]string, error)
	AddKeyFunc      func(string) error
	RemoveKeyFunc   func(string) error
	ImportKeyFunc   func(string) error

	// Repairer
	RemoveLockFunc func() error

	// GroupManager
	ListGroupsFunc       func() ([]string, error)
	GetGroupPackagesFunc func(group string) ([]string, error)

	// DatabaseManager
	SetInstallReasonFunc func(pkg string, reason core.InstallReason) error
	CheckDatabaseFunc    func() error

	// Exporter
	ListExplicitPackagesFunc func() ([]string, error)

	// SecurityScanner
	AuditFunc func() ([]string, error)

	// Differ
	DiffFunc func() ([]core.PackageDiff, error)
}

func (m *MockBackend) Name() string {
	if m.NameFunc != nil {
		return m.NameFunc()
	}
	return "mock"
}

func (m *MockBackend) IsAvailable() bool {
	if m.IsAvailableFunc != nil {
		return m.IsAvailableFunc()
	}
	return true
}

func (m *MockBackend) Sync() error {
	if m.SyncFunc != nil {
		return m.SyncFunc()
	}
	return nil
}

func (m *MockBackend) Install(pkgs []string, opts core.InstallOptions) error {
	if m.InstallFunc != nil {
		return m.InstallFunc(pkgs, opts)
	}
	return nil
}

func (m *MockBackend) Remove(pkgs []string, opts core.RemoveOptions) error {
	if m.RemoveFunc != nil {
		return m.RemoveFunc(pkgs, opts)
	}
	return nil
}

func (m *MockBackend) Upgrade(pkgs []string, opts core.UpgradeOptions) error {
	if m.UpgradeFunc != nil {
		return m.UpgradeFunc(pkgs, opts)
	}
	return nil
}

func (m *MockBackend) IsInstalled(pkgName string) bool {
	if m.IsInstalledFunc != nil {
		return m.IsInstalledFunc(pkgName)
	}
	return false
}

func (m *MockBackend) Info(pkgName string) (*core.PackageInfo, error) {
	if m.InfoFunc != nil {
		return m.InfoFunc(pkgName)
	}
	return nil, core.ErrPackageNotFound
}

func (m *MockBackend) GetInstalledPackages() ([]core.PackageInfo, error) {
	if m.InstalledPkgsFunc != nil {
		return m.InstalledPkgsFunc()
	}
	return nil, nil
}

func (m *MockBackend) GetPackageFiles(pkgName string) ([]string, error) {
	if m.PkgFilesFunc != nil {
		return m.PkgFilesFunc(pkgName)
	}
	return nil, nil
}

func (m *MockBackend) InstallLocal(path string, opts core.InstallOptions) error {
	if m.InstallLocalFunc != nil {
		return m.InstallLocalFunc(path, opts)
	}
	return nil
}

func (m *MockBackend) Search(query string) ([]core.PackageInfo, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(query)
	}
	return nil, nil
}

// Maintainer implementation

func (m *MockBackend) CleanCache(opts core.CleanOptions) error {
	if m.CleanCacheFunc != nil {
		return m.CleanCacheFunc(opts)
	}
	return nil
}

func (m *MockBackend) ListOrphans() ([]string, error) {
	if m.ListOrphansFunc != nil {
		return m.ListOrphansFunc()
	}
	return nil, nil
}

func (m *MockBackend) RemoveOrphans(pkgs []string) error {
	if m.RemoveOrphansFunc != nil {
		return m.RemoveOrphansFunc(pkgs)
	}
	return nil
}

// FileProvider implementation

func (m *MockBackend) GetFileOwner(path string) (string, error) {
	if m.GetFileOwnerFunc != nil {
		return m.GetFileOwnerFunc(path)
	}
	return "", nil
}

func (m *MockBackend) SearchFiles(query string) ([]string, error) {
	if m.SearchFilesFunc != nil {
		return m.SearchFilesFunc(query)
	}
	return nil, nil
}

// Verifier implementation

func (m *MockBackend) VerifyAll() (map[string][]string, error) {
	if m.VerifyAllFunc != nil {
		return m.VerifyAllFunc()
	}
	return nil, nil // Or add VerifyAllFunc
}

// Add hook definition field to struct previously

func (m *MockBackend) VerifyPackage(pkgName string) ([]string, error) {
	if m.VerifyPackageFunc != nil {
		return m.VerifyPackageFunc(pkgName)
	}
	return nil, nil
}

// Builder implementation

func (m *MockBackend) Build(dir string, opts core.BuildOptions) error {
	if m.BuildFunc != nil {
		return m.BuildFunc(dir, opts)
	}
	return nil
}

// KeyManager implementation

func (m *MockBackend) InitKeyring() error {
	if m.InitKeyringFunc != nil {
		return m.InitKeyringFunc()
	}
	return nil
}

func (m *MockBackend) RefreshKeys() error {
	if m.RefreshKeysFunc != nil {
		return m.RefreshKeysFunc()
	}
	return nil
}

func (m *MockBackend) ListKeys() ([]string, error) {
	if m.ListKeysFunc != nil {
		return m.ListKeysFunc()
	}
	return nil, nil
}

func (m *MockBackend) AddKey(keyID string) error {
	if m.AddKeyFunc != nil {
		return m.AddKeyFunc(keyID)
	}
	return nil
}

func (m *MockBackend) RemoveKey(keyID string) error {
	if m.RemoveKeyFunc != nil {
		return m.RemoveKeyFunc(keyID)
	}
	return nil
}

func (m *MockBackend) ImportKey(path string) error {
	if m.ImportKeyFunc != nil {
		return m.ImportKeyFunc(path)
	}
	return nil
}

// Repairer implementation

func (m *MockBackend) RemoveLock() error {
	if m.RemoveLockFunc != nil {
		return m.RemoveLockFunc()
	}
	return nil
}

// GroupManager implementation

func (m *MockBackend) ListGroups() ([]string, error) {
	if m.ListGroupsFunc != nil {
		return m.ListGroupsFunc()
	}
	return nil, nil
}

func (m *MockBackend) GetGroupPackages(group string) ([]string, error) {
	if m.GetGroupPackagesFunc != nil {
		return m.GetGroupPackagesFunc(group)
	}
	return nil, nil
}

// DatabaseManager implementation

func (m *MockBackend) SetInstallReason(pkg string, reason core.InstallReason) error {
	if m.SetInstallReasonFunc != nil {
		return m.SetInstallReasonFunc(pkg, reason)
	}
	return nil
}

func (m *MockBackend) CheckDatabase() error {
	if m.CheckDatabaseFunc != nil {
		return m.CheckDatabaseFunc()
	}
	return nil
}

// Exporter implementation

func (m *MockBackend) ListExplicitPackages() ([]string, error) {
	if m.ListExplicitPackagesFunc != nil {
		return m.ListExplicitPackagesFunc()
	}
	return nil, nil
}

// SecurityScanner implementation

func (m *MockBackend) Audit() ([]string, error) {
	if m.AuditFunc != nil {
		return m.AuditFunc()
	}
	return nil, nil
}

// Differ implementation

func (m *MockBackend) Diff() ([]core.PackageDiff, error) {
	if m.DiffFunc != nil {
		return m.DiffFunc()
	}
	return nil, nil
}
