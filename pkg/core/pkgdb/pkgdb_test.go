package pkgdb_test

import (
	"testing"

	"github.com/theshedman/shedman/pkg/shedman/pkgdb"
)

func TestPackageInfo_HasRequiredFields(t *testing.T) {
	pkg := pkgdb.PackageInfo{
		Name:        "neovim",
		Version:     "0.10.0-1",
		Description: "Vim-fork focused on extensibility and usability",
		Source:      pkgdb.SourceOfficial,
	}

	if pkg.Name != "neovim" {
		t.Errorf("Expected Name 'neovim', got '%s'", pkg.Name)
	}
	if pkg.Version != "0.10.0-1" {
		t.Errorf("Expected Version '0.10.0-1', got '%s'", pkg.Version)
	}
	if pkg.Source != pkgdb.SourceOfficial {
		t.Errorf("Expected Source 'official', got '%s'", pkg.Source)
	}
}

func TestPackageSource_Constants(t *testing.T) {
	if pkgdb.SourceOfficial != "official" {
		t.Error("SourceOfficial should be 'official'")
	}
	if pkgdb.SourceAUR != "aur" {
		t.Error("SourceAUR should be 'aur'")
	}
	if pkgdb.SourceShedOS != "shedos" {
		t.Error("SourceShedOS should be 'shedos'")
	}
}

func TestPackageDB_Search_Interface(t *testing.T) {
	// Create a mock that implements PackageDB
	mock := &mockPackageDB{
		packages: []pkgdb.PackageInfo{
			{Name: "neovim", Version: "0.10.0", Source: pkgdb.SourceOfficial},
			{Name: "neovim-nightly", Version: "0.11.0", Source: pkgdb.SourceAUR},
		},
	}

	// Verify it satisfies the interface
	var db pkgdb.PackageDB = mock

	results, err := db.Search("neovim")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

func TestPackageDB_GetInfo_Interface(t *testing.T) {
	mock := &mockPackageDB{
		packages: []pkgdb.PackageInfo{
			{Name: "neovim", Version: "0.10.0", Source: pkgdb.SourceOfficial},
		},
	}

	var db pkgdb.PackageDB = mock

	info, err := db.GetInfo("neovim")
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}

	if info.Name != "neovim" {
		t.Errorf("Expected 'neovim', got '%s'", info.Name)
	}
}

// Mock implementation for testing
type mockPackageDB struct {
	packages []pkgdb.PackageInfo
}

func (m *mockPackageDB) Search(query string) ([]pkgdb.PackageInfo, error) {
	return m.packages, nil
}

func (m *mockPackageDB) GetInfo(name string) (*pkgdb.PackageInfo, error) {
	for _, p := range m.packages {
		if p.Name == name {
			return &p, nil
		}
	}
	return nil, nil
}
