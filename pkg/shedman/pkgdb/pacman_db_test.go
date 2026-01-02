package pkgdb_test

import (
	"testing"

	"github.com/theshedman/shedman/pkg/shedman/pkgdb"
)

func TestPacmanDB_NewPacmanDB(t *testing.T) {
	db := pkgdb.NewPacmanDB()
	if db == nil {
		t.Fatal("NewPacmanDB should return non-nil")
	}
}

func TestPacmanDB_ImplementsPackageDB(t *testing.T) {
	db := pkgdb.NewPacmanDB()

	// Verify it satisfies the PackageDB interface
	var _ pkgdb.PackageDB = db
}

func TestPacmanDB_Search(t *testing.T) {
	db := pkgdb.NewPacmanDB()

	var executedCmd []string
	db.SetExecutor(func(cmd []string) (string, error) {
		executedCmd = cmd
		// Return fake pacman output
		return "neovim 0.10.0-1\n    Vim fork focused on extensibility and usability\n", nil
	})

	results, err := db.Search("neovim")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Should call pacman -Ss
	if len(executedCmd) < 2 || executedCmd[0] != "pacman" || executedCmd[1] != "-Ss" {
		t.Errorf("Expected 'pacman -Ss', got %v", executedCmd)
	}

	if len(results) == 0 {
		t.Error("Expected at least one result")
	}
}

func TestPacmanDB_GetInfo(t *testing.T) {
	db := pkgdb.NewPacmanDB()

	db.SetExecutor(func(cmd []string) (string, error) {
		// Return fake pacman -Si output
		return `Name            : neovim
Version         : 0.10.0-1
Description     : Vim fork focused on extensibility and usability
Depends On      : luajit  msgpack-c  unibilium
Optional Deps   : python-pynvim: for Python plugin support
Provides        : vim
Conflicts       : None
Download Size   : 10.5 MiB
Installed Size  : 45.2 MiB
`, nil
	})

	info, err := db.GetInfo("neovim")
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}

	if info == nil {
		t.Fatal("Expected package info, got nil")
	}

	if info.Name != "neovim" {
		t.Errorf("Expected name 'neovim', got '%s'", info.Name)
	}
	if info.Version != "0.10.0-1" {
		t.Errorf("Expected version '0.10.0-1', got '%s'", info.Version)
	}
	if len(info.Depends) == 0 {
		t.Error("Expected dependencies")
	}
}

func TestPacmanDB_GetInfo_NotFound(t *testing.T) {
	db := pkgdb.NewPacmanDB()

	db.SetExecutor(func(cmd []string) (string, error) {
		return "error: package 'nonexistent' was not found\n", nil
	})

	info, err := db.GetInfo("nonexistent")
	if err != nil {
		t.Fatalf("GetInfo should not error for not found: %v", err)
	}

	if info != nil {
		t.Error("Expected nil for non-existent package")
	}
}

func TestPacmanDB_IsInstalled(t *testing.T) {
	db := pkgdb.NewPacmanDB()

	db.SetExecutor(func(cmd []string) (string, error) {
		if cmd[1] == "-Q" {
			return "neovim 0.10.0-1\n", nil
		}
		return "", nil
	})

	installed := db.IsInstalled("neovim")
	if !installed {
		t.Error("Expected neovim to be installed")
	}
}

func TestPacmanDB_ParsePacmanInfo(t *testing.T) {
	output := `Name            : git
Version         : 2.43.0-1
Description     : The fast distributed version control system
Depends On      : curl  expat  perl  openssl
Optional Deps   : libsecret: credential storage
Provides        : git-core
Conflicts       : git-core
Download Size   : 8.5 MiB
Installed Size  : 35.0 MiB
`
	info := pkgdb.ParsePacmanInfo(output)

	if info.Name != "git" {
		t.Errorf("Expected name 'git', got '%s'", info.Name)
	}
	if info.Version != "2.43.0-1" {
		t.Errorf("Expected version '2.43.0-1', got '%s'", info.Version)
	}
	if len(info.Depends) != 4 {
		t.Errorf("Expected 4 dependencies, got %d", len(info.Depends))
	}
	if len(info.Provides) != 1 || info.Provides[0] != "git-core" {
		t.Errorf("Expected provides [git-core], got %v", info.Provides)
	}
	if len(info.Conflicts) != 1 || info.Conflicts[0] != "git-core" {
		t.Errorf("Expected conflicts [git-core], got %v", info.Conflicts)
	}

	// Verify size parsing - 8.5 MiB = 8.5 * 1024 * 1024 = 8912896 bytes
	expectedSize := int64(8912896)
	if info.Size != expectedSize {
		t.Errorf("Expected download size %d, got %d", expectedSize, info.Size)
	}

	// 35.0 MiB = 35 * 1024 * 1024 = 36700160 bytes
	expectedInstalled := int64(36700160)
	if info.InstalledSize != expectedInstalled {
		t.Errorf("Expected installed size %d, got %d", expectedInstalled, info.InstalledSize)
	}
}

func TestPacmanDB_ParseSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"10.5 MiB", 11010048},
		{"8.5 MiB", 8912896},
		{"35.0 MiB", 36700160},
		{"1.0 GiB", 1073741824},
		{"512 KiB", 524288},
		{"100 B", 100},
		{"", 0},
		{"invalid", 0},
	}

	for _, tc := range tests {
		result := pkgdb.ParseSize(tc.input)
		if result != tc.expected {
			t.Errorf("ParseSize(%q) = %d, expected %d", tc.input, result, tc.expected)
		}
	}
}

func TestPacmanDB_CommandCheck(t *testing.T) {
	// PacmanDB should have a way to check if pacman exists
	db := pkgdb.NewPacmanDB()

	// This just verifies the check method exists and is callable
	_ = db.IsPacmanAvailable()
}
