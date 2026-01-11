package pacman

import (
	"testing"

	libalpm "github.com/Jguer/go-alpm/v2"
	"github.com/theshedman/shedman/internal/alpm"
	"github.com/theshedman/shedman/pkg/core"
)

// Compile-time interface checks to ensure AlpmBackend implements all required capabilities
var (
	_ core.Backend         = (*AlpmBackend)(nil)
	_ core.PackageManager  = (*AlpmBackend)(nil)
	_ core.Searchable      = (*AlpmBackend)(nil)
	_ core.Informer        = (*AlpmBackend)(nil)
	_ core.Upgradable      = (*AlpmBackend)(nil)
	_ core.LocalInstaller  = (*AlpmBackend)(nil)
	_ core.FileProvider    = (*AlpmBackend)(nil)
	_ core.OfficialBackend = (*AlpmBackend)(nil)
)

func TestAlpmBackend_Search(t *testing.T) {
	vimPkg := &alpm.MockAlpmPackage{NameVal: "vim", VersionVal: "9.0.0", DescriptionVal: "Vi Improved"}
	neovimPkg := &alpm.MockAlpmPackage{NameVal: "neovim", VersionVal: "0.10.0", DescriptionVal: "Fork of Vim"}

	mockDB := &alpm.MockAlpmDB{
		NameVal: "extra",
		SearchFn: func(targets []string) alpm.AlpmPackageList {
			return &alpm.MockAlpmPackageList{Packages: []alpm.AlpmPackage{vimPkg, neovimPkg}}
		},
	}

	mockHandle := &alpm.MockAlpmHandle{
		SyncDBs: &alpm.MockAlpmDBList{Dbs: []alpm.AlpmDB{mockDB}},
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
	vimPkg := &alpm.MockAlpmPackage{NameVal: "vim", VersionVal: "9.0.0", DescriptionVal: "Vi Improved"}

	mockDB := &alpm.MockAlpmDB{
		NameVal:  "extra",
		Packages: map[string]alpm.AlpmPackage{"vim": vimPkg},
	}

	mockHandle := &alpm.MockAlpmHandle{
		SyncDBs: &alpm.MockAlpmDBList{Dbs: []alpm.AlpmDB{mockDB}},
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
	mockDB := &alpm.MockAlpmDB{
		NameVal:  "extra",
		Packages: map[string]alpm.AlpmPackage{},
	}

	mockHandle := &alpm.MockAlpmHandle{
		SyncDBs: &alpm.MockAlpmDBList{Dbs: []alpm.AlpmDB{mockDB}},
	}

	b := NewAlpmBackendWithHandle(mockHandle)
	_, err := b.Info("nonexistent")

	if err != core.ErrPackageNotFound {
		t.Errorf("Expected ErrPackageNotFound, got %v", err)
	}
}

func TestAlpmBackend_GetInstalledPackages(t *testing.T) {
	vimPkg := &alpm.MockAlpmPackage{NameVal: "vim", VersionVal: "9.0.0"}
	bashPkg := &alpm.MockAlpmPackage{NameVal: "bash", VersionVal: "5.2"}

	mockLocalDB := &alpm.MockAlpmDB{
		NameVal:     "local",
		PkgCacheVal: &alpm.MockAlpmPackageList{Packages: []alpm.AlpmPackage{vimPkg, bashPkg}},
	}

	mockHandle := &alpm.MockAlpmHandle{
		LocalDB: mockLocalDB,
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
	vimPkg := &alpm.MockAlpmPackage{NameVal: "vim", VersionVal: "9.0.0"}

	mockLocalDB := &alpm.MockAlpmDB{
		NameVal:  "local",
		Packages: map[string]alpm.AlpmPackage{"vim": vimPkg},
	}

	mockHandle := &alpm.MockAlpmHandle{
		LocalDB: mockLocalDB,
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
	vimPkg := &alpm.MockAlpmPackage{
		NameVal:    "vim",
		VersionVal: "9.0.0",
		FilesVal:   []libalpm.File{{Name: "/usr/bin/vim", Size: 1024, Mode: 0755}, {Name: "/usr/share/vim/vimrc", Size: 256, Mode: 0644}},
	}

	mockLocalDB := &alpm.MockAlpmDB{
		NameVal:  "local",
		Packages: map[string]alpm.AlpmPackage{"vim": vimPkg},
	}

	mockHandle := &alpm.MockAlpmHandle{
		LocalDB: mockLocalDB,
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
	mockLocalDB := &alpm.MockAlpmDB{
		NameVal:  "local",
		Packages: map[string]alpm.AlpmPackage{},
	}

	mockHandle := &alpm.MockAlpmHandle{
		LocalDB: mockLocalDB,
	}

	b := NewAlpmBackendWithHandle(mockHandle)
	_, err := b.GetPackageFiles("nonexistent")

	if err != core.ErrPackageNotFound {
		t.Errorf("Expected ErrPackageNotFound, got %v", err)
	}
}


func TestAlpmBackend_InstallLocal(t *testing.T) {
	mockExecutor := &MockExecutor{}
	b := &AlpmBackend{
		executor: mockExecutor,
		sudoPath: "sudo",
	}

	err := b.InstallLocal("test-pkg.pkg.tar.zst", core.InstallOptions{NoConfirm: true})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(mockExecutor.RunCalls) != 1 {
		t.Fatalf("Expected 1 call, got %d", len(mockExecutor.RunCalls))
	}

	call := mockExecutor.RunCalls[0]

	// Call format: [sudo, pacman, -U, pkg, --noconfirm]
	if len(call) < 3 {
		t.Fatalf("Call too short: %v", call)
	}

	if call[0] != "sudo" {
		t.Errorf("Expected command 'sudo', got '%s'", call[0])
	}
	if call[1] != "pacman" {
		t.Errorf("Expected 'pacman' as second arg, got '%s'", call[1])
	}

	// Verify flags/args are present
	expectedArgs := []string{"-U", "test-pkg.pkg.tar.zst", "--noconfirm"}
	pacmanArgs := call[2:] // args passed to pacman

	for _, expected := range expectedArgs {
		found := false
		for _, actual := range pacmanArgs {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected arg '%s' not found in %v", expected, pacmanArgs)
		}
	}
}
