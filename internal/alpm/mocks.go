package alpm

import "github.com/Jguer/go-alpm/v2"

// MockAlpmHandle implements AlpmHandle for testing.
type MockAlpmHandle struct {
	LocalDB  AlpmDB
	SyncDBs  AlpmDBList
	Released bool
	RootPath string
	DbPath   string
}

func (m *MockAlpmHandle) LocalDb() AlpmDB                      { return m.LocalDB }
func (m *MockAlpmHandle) SyncDbs() AlpmDBList                  { return m.SyncDBs }
func (m *MockAlpmHandle) Release() error                       { m.Released = true; return nil }
func (m *MockAlpmHandle) Root() string                         { return m.RootPath }
func (m *MockAlpmHandle) DBPath() string                       { return m.DbPath }
func (m *MockAlpmHandle) TransInit(flags alpm.TransFlag) error { return nil }
func (m *MockAlpmHandle) TransPrepare() error                  { return nil }
func (m *MockAlpmHandle) TransCommit() error                   { return nil }
func (m *MockAlpmHandle) TransRelease() error                  { return nil }
func (m *MockAlpmHandle) AddPkg(pkg AlpmPackage) error         { return nil }
func (m *MockAlpmHandle) RemovePkg(pkg AlpmPackage) error      { return nil }
func (m *MockAlpmHandle) SyncDBsForce() AlpmDBList             { return m.SyncDBs }
func (m *MockAlpmHandle) GetRawHandle() *alpm.Handle           { return nil }
func (m *MockAlpmHandle) IsIgnored(pkgName string) bool        { return false }

// MockAlpmDB implements AlpmDB for testing.
type MockAlpmDB struct {
	NameVal string
	// Packages map[string]AlpmPackage // Can't easily export map access without getter setter or public field
	// But tests assign to struct literal. So define exported fields.
	Packages    map[string]AlpmPackage
	SearchFn    func([]string) AlpmPackageList
	PkgCacheVal AlpmPackageList
}

func (m *MockAlpmDB) Name() string { return m.NameVal }

func (m *MockAlpmDB) Pkg(name string) AlpmPackage {
	if m.Packages == nil {
		return nil
	}
	return m.Packages[name]
}

func (m *MockAlpmDB) PkgCache() AlpmPackageList {
	return m.PkgCacheVal
}

func (m *MockAlpmDB) Search(targets []string) AlpmPackageList {
	if m.SearchFn != nil {
		return m.SearchFn(targets)
	}
	return &MockAlpmPackageList{Packages: nil}
}

// MockAlpmDBList implements AlpmDBList for testing.
type MockAlpmDBList struct {
	Dbs []AlpmDB
}

func (m *MockAlpmDBList) ForEach(f func(AlpmDB) error) error {
	for _, db := range m.Dbs {
		if err := f(db); err != nil {
			return err
		}
	}
	return nil
}

func (m *MockAlpmDBList) Slice() []AlpmDB {
	return m.Dbs
}

// MockAlpmPackage implements AlpmPackage for testing.
type MockAlpmPackage struct {
	NameVal        string
	VersionVal     string
	DescriptionVal string
	DependsVal     []string
	OptDependsVal  []string
	ProvidesVal    []string
	ConflictsVal   []string
	SizeVal        int64
	IsizeVal       int64
	FilesVal       []alpm.File
	DbVal          AlpmDB
}

func (m *MockAlpmPackage) Name() string            { return m.NameVal }
func (m *MockAlpmPackage) Version() string         { return m.VersionVal }
func (m *MockAlpmPackage) Description() string     { return m.DescriptionVal }
func (m *MockAlpmPackage) Depends() AlpmDependList { return &MockAlpmDependList{Deps: m.DependsVal} }
func (m *MockAlpmPackage) OptionalDepends() AlpmDependList {
	return &MockAlpmDependList{Deps: m.OptDependsVal}
}
func (m *MockAlpmPackage) Provides() AlpmDependList { return &MockAlpmDependList{Deps: m.ProvidesVal} }
func (m *MockAlpmPackage) Conflicts() AlpmDependList {
	return &MockAlpmDependList{Deps: m.ConflictsVal}
}
func (m *MockAlpmPackage) Size() int64        { return m.SizeVal }
func (m *MockAlpmPackage) ISize() int64       { return m.IsizeVal }
func (m *MockAlpmPackage) Files() []alpm.File { return m.FilesVal }
func (m *MockAlpmPackage) DB() AlpmDB         { return m.DbVal }

// MockAlpmPackageList implements AlpmPackageList for testing.
type MockAlpmPackageList struct {
	Packages []AlpmPackage
}

func (m *MockAlpmPackageList) ForEach(f func(AlpmPackage) error) error {
	for _, pkg := range m.Packages {
		if err := f(pkg); err != nil {
			return err
		}
	}
	return nil
}

func (m *MockAlpmPackageList) Slice() []AlpmPackage {
	return m.Packages
}

func (m *MockAlpmPackageList) Len() int {
	return len(m.Packages)
}

// MockAlpmDependList implements AlpmDependList for testing.
type MockAlpmDependList struct {
	Deps []string
}

func (m *MockAlpmDependList) Slice() []string {
	return m.Deps
}
