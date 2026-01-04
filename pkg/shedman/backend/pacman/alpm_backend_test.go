package pacman

import (
	"testing"

	"github.com/Jguer/go-alpm/v2"
	"github.com/theshedman/shedman/pkg/shedman/backend"
)

func TestAlpmBackend_Search(t *testing.T) {
	vimPkg := &MockAlpmPackage{name: "vim", version: "9.0.0", description: "Vi Improved"}
	neovimPkg := &MockAlpmPackage{name: "neovim", version: "0.10.0", description: "Fork of Vim"}

	mockDB := &MockAlpmDB{
		name: "extra",
		searchFn: func(targets []string) AlpmPackageList {
			return &MockAlpmPackageList{packages: []AlpmPackage{vimPkg, neovimPkg}}
		},
	}

	mockHandle := &MockAlpmHandle{
		syncDBs: &MockAlpmDBList{dbs: []AlpmDB{mockDB}},
	}

	b := NewAlpmBackendWithHandle(mockHandle)
	results, err := b.Search("vim")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
	if results[0].Name != "vim" {
		t.Errorf("Expected first result 'vim', got '%s'", results[0].Name)
	}
}

func TestAlpmBackend_Info(t *testing.T) {
	vimPkg := &MockAlpmPackage{name: "vim", version: "9.0.0", description: "Vi Improved"}

	mockDB := &MockAlpmDB{
		name:     "extra",
		packages: map[string]AlpmPackage{"vim": vimPkg},
	}

	mockHandle := &MockAlpmHandle{
		syncDBs: &MockAlpmDBList{dbs: []AlpmDB{mockDB}},
	}

	b := NewAlpmBackendWithHandle(mockHandle)
	info, err := b.Info("vim")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if info == nil {
		t.Fatal("Expected info, got nil")
	}
	if info.Name != "vim" {
		t.Errorf("Expected 'vim', got '%s'", info.Name)
	}
}

func TestAlpmBackend_Info_NotFound(t *testing.T) {
	mockDB := &MockAlpmDB{
		name:     "extra",
		packages: map[string]AlpmPackage{},
	}

	mockHandle := &MockAlpmHandle{
		syncDBs: &MockAlpmDBList{dbs: []AlpmDB{mockDB}},
	}

	b := NewAlpmBackendWithHandle(mockHandle)
	_, err := b.Info("nonexistent")

	if err != backend.ErrPackageNotFound {
		t.Errorf("Expected ErrPackageNotFound, got %v", err)
	}
}

func TestAlpmBackend_GetInstalledPackages(t *testing.T) {
	vimPkg := &MockAlpmPackage{name: "vim", version: "9.0.0"}
	bashPkg := &MockAlpmPackage{name: "bash", version: "5.2"}

	mockLocalDB := &MockAlpmDB{
		name:     "local",
		pkgCache: &MockAlpmPackageList{packages: []AlpmPackage{vimPkg, bashPkg}},
	}

	mockHandle := &MockAlpmHandle{
		localDB: mockLocalDB,
	}

	b := NewAlpmBackendWithHandle(mockHandle)
	packages, err := b.GetInstalledPackages()

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(packages) != 2 {
		t.Errorf("Expected 2 packages, got %d", len(packages))
	}
}

func TestAlpmBackend_IsInstalled(t *testing.T) {
	vimPkg := &MockAlpmPackage{name: "vim", version: "9.0.0"}

	mockLocalDB := &MockAlpmDB{
		name:     "local",
		packages: map[string]AlpmPackage{"vim": vimPkg},
	}

	mockHandle := &MockAlpmHandle{
		localDB: mockLocalDB,
	}

	b := NewAlpmBackendWithHandle(mockHandle)

	if !b.IsInstalled("vim") {
		t.Error("Expected vim to be installed")
	}
	if b.IsInstalled("nonexistent") {
		t.Error("Expected nonexistent to not be installed")
	}
}

func TestAlpmBackend_Name(t *testing.T) {
	b := &AlpmBackend{}
	if b.Name() != "pacman" {
		t.Errorf("Expected 'pacman', got '%s'", b.Name())
	}
}

func TestAlpmBackend_DistroFamily(t *testing.T) {
	b := &AlpmBackend{}
	if b.DistroFamily() != "arch" {
		t.Errorf("Expected 'arch', got '%s'", b.DistroFamily())
	}
}

func TestAlpmBackend_GetPackageFiles(t *testing.T) {
	vimPkg := &MockAlpmPackage{
		name:    "vim",
		version: "9.0.0",
		files:   []alpm.File{{Name: "/usr/bin/vim"}, {Name: "/usr/share/vim/vimrc"}},
	}

	mockLocalDB := &MockAlpmDB{
		name:     "local",
		packages: map[string]AlpmPackage{"vim": vimPkg},
	}

	mockHandle := &MockAlpmHandle{
		localDB: mockLocalDB,
	}

	b := NewAlpmBackendWithHandle(mockHandle)
	files, err := b.GetPackageFiles("vim")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}
	if files[0] != "/usr/bin/vim" {
		t.Errorf("Expected '/usr/bin/vim', got '%s'", files[0])
	}
}

func TestAlpmBackend_GetPackageFiles_NotFound(t *testing.T) {
	mockLocalDB := &MockAlpmDB{
		name:     "local",
		packages: map[string]AlpmPackage{},
	}

	mockHandle := &MockAlpmHandle{
		localDB: mockLocalDB,
	}

	b := NewAlpmBackendWithHandle(mockHandle)
	_, err := b.GetPackageFiles("nonexistent")

	if err != backend.ErrPackageNotFound {
		t.Errorf("Expected ErrPackageNotFound, got %v", err)
	}
}
