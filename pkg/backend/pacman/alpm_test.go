package pacman

import (
	"testing"

	"github.com/Jguer/go-alpm/v2"
	"github.com/theshedman/shedman/pkg/core/pkgdb"
)

// MockAlpmHandle implements AlpmHandle for testing.
type MockAlpmHandle struct {
	localDB  AlpmDB
	syncDBs  AlpmDBList
	released bool
	root     string
	dbPath   string
}

func (m *MockAlpmHandle) LocalDb() AlpmDB                      { return m.localDB }
func (m *MockAlpmHandle) SyncDbs() AlpmDBList                  { return m.syncDBs }
func (m *MockAlpmHandle) Release() error                       { m.released = true; return nil }
func (m *MockAlpmHandle) Root() string                         { return m.root }
func (m *MockAlpmHandle) DBPath() string                       { return m.dbPath }
func (m *MockAlpmHandle) TransInit(flags alpm.TransFlag) error { return nil }
func (m *MockAlpmHandle) TransPrepare() error                  { return nil }
func (m *MockAlpmHandle) TransCommit() error                   { return nil }
func (m *MockAlpmHandle) TransRelease() error                  { return nil }
func (m *MockAlpmHandle) AddPkg(pkg AlpmPackage) error         { return nil }
func (m *MockAlpmHandle) RemovePkg(pkg AlpmPackage) error      { return nil }
func (m *MockAlpmHandle) SyncDBsForce() AlpmDBList             { return m.syncDBs }
func (m *MockAlpmHandle) GetRawHandle() *alpm.Handle           { return nil }
func (m *MockAlpmHandle) IsIgnored(pkgName string) bool        { return false }

// MockAlpmDB implements AlpmDB for testing.
type MockAlpmDB struct {
	name     string
	packages map[string]AlpmPackage
	searchFn func([]string) AlpmPackageList
	pkgCache AlpmPackageList
}

func (m *MockAlpmDB) Name() string { return m.name }

func (m *MockAlpmDB) Pkg(name string) AlpmPackage {
	if m.packages == nil {
		return nil
	}
	return m.packages[name]
}

func (m *MockAlpmDB) PkgCache() AlpmPackageList {
	return m.pkgCache
}

func (m *MockAlpmDB) Search(targets []string) AlpmPackageList {
	if m.searchFn != nil {
		return m.searchFn(targets)
	}
	return &MockAlpmPackageList{packages: nil}
}

// MockAlpmDBList implements AlpmDBList for testing.
type MockAlpmDBList struct {
	dbs []AlpmDB
}

func (m *MockAlpmDBList) ForEach(f func(AlpmDB) error) error {
	for _, db := range m.dbs {
		if err := f(db); err != nil {
			return err
		}
	}
	return nil
}

func (m *MockAlpmDBList) Slice() []AlpmDB {
	return m.dbs
}

// MockAlpmPackage implements AlpmPackage for testing.
type MockAlpmPackage struct {
	name        string
	version     string
	description string
	depends     []string
	optDepends  []string
	provides    []string
	conflicts   []string
	size        int64
	isize       int64
	files       []alpm.File
	db          AlpmDB
}

func (m *MockAlpmPackage) Name() string            { return m.name }
func (m *MockAlpmPackage) Version() string         { return m.version }
func (m *MockAlpmPackage) Description() string     { return m.description }
func (m *MockAlpmPackage) Depends() AlpmDependList { return &MockAlpmDependList{deps: m.depends} }
func (m *MockAlpmPackage) OptionalDepends() AlpmDependList {
	return &MockAlpmDependList{deps: m.optDepends}
}
func (m *MockAlpmPackage) Provides() AlpmDependList  { return &MockAlpmDependList{deps: m.provides} }
func (m *MockAlpmPackage) Conflicts() AlpmDependList { return &MockAlpmDependList{deps: m.conflicts} }
func (m *MockAlpmPackage) Size() int64               { return m.size }
func (m *MockAlpmPackage) ISize() int64              { return m.isize }
func (m *MockAlpmPackage) Files() []alpm.File        { return m.files }
func (m *MockAlpmPackage) DB() AlpmDB                { return m.db }

// MockAlpmPackageList implements AlpmPackageList for testing.
type MockAlpmPackageList struct {
	packages []AlpmPackage
}

func (m *MockAlpmPackageList) ForEach(f func(AlpmPackage) error) error {
	for _, pkg := range m.packages {
		if err := f(pkg); err != nil {
			return err
		}
	}
	return nil
}

func (m *MockAlpmPackageList) Slice() []AlpmPackage {
	return m.packages
}

func (m *MockAlpmPackageList) Len() int {
	return len(m.packages)
}

// MockAlpmDependList implements AlpmDependList for testing.
type MockAlpmDependList struct {
	deps []string
}

func (m *MockAlpmDependList) Slice() []string {
	return m.deps
}

// --- Tests ---

func TestPackageToInfo(t *testing.T) {
	pkg := &MockAlpmPackage{
		name:        "neovim",
		version:     "0.10.0-1",
		description: "Fork of Vim",
		depends:     []string{"libuv", "msgpack-c"},
		optDepends:  []string{"python: for plugins"},
		provides:    []string{"vim"},
		conflicts:   []string{"vim"},
		size:        1024000,
		isize:       2048000,
	}

	info := PackageToInfo(pkg)

	if info.Name != "neovim" {
		t.Errorf("Expected name 'neovim', got '%s'", info.Name)
	}
	if info.Version != "0.10.0-1" {
		t.Errorf("Expected version '0.10.0-1', got '%s'", info.Version)
	}
	if info.Description != "Fork of Vim" {
		t.Errorf("Expected description 'Fork of Vim', got '%s'", info.Description)
	}
	if len(info.Depends) != 2 {
		t.Errorf("Expected 2 depends, got %d", len(info.Depends))
	}
	if info.Source != pkgdb.SourceOfficial {
		t.Errorf("Expected source 'official', got '%s'", info.Source)
	}
}

func TestMockAlpmDB_Pkg(t *testing.T) {
	mockPkg := &MockAlpmPackage{name: "vim", version: "9.0.0"}
	mockDB := &MockAlpmDB{
		name:     "core",
		packages: map[string]AlpmPackage{"vim": mockPkg},
	}

	pkg := mockDB.Pkg("vim")
	if pkg == nil {
		t.Fatal("Expected package, got nil")
	}
	if pkg.Name() != "vim" {
		t.Errorf("Expected 'vim', got '%s'", pkg.Name())
	}

	notFound := mockDB.Pkg("nonexistent")
	if notFound != nil {
		t.Error("Expected nil for nonexistent package")
	}
}

func TestMockAlpmDB_Search(t *testing.T) {
	vimPkg := &MockAlpmPackage{name: "vim", version: "9.0.0"}
	neovimPkg := &MockAlpmPackage{name: "neovim", version: "0.10.0"}

	mockDB := &MockAlpmDB{
		name: "extra",
		searchFn: func(targets []string) AlpmPackageList {
			// Simple mock: return both packages for any search
			return &MockAlpmPackageList{packages: []AlpmPackage{vimPkg, neovimPkg}}
		},
	}

	results := mockDB.Search([]string{"vim"})
	if results.Len() != 2 {
		t.Errorf("Expected 2 results, got %d", results.Len())
	}
}

func TestMockAlpmDBList_ForEach(t *testing.T) {
	db1 := &MockAlpmDB{name: "core"}
	db2 := &MockAlpmDB{name: "extra"}

	dbList := &MockAlpmDBList{dbs: []AlpmDB{db1, db2}}

	var names []string
	err := dbList.ForEach(func(db AlpmDB) error {
		names = append(names, db.Name())
		return nil
	})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(names) != 2 {
		t.Errorf("Expected 2 databases, got %d", len(names))
	}
}

func TestMockAlpmHandle_Release(t *testing.T) {
	handle := &MockAlpmHandle{}

	if handle.released {
		t.Error("Handle should not be released initially")
	}

	err := handle.Release()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !handle.released {
		t.Error("Handle should be released after Release()")
	}
}
