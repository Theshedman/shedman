package core

import (
	"testing"

)

func TestPackageInfo_HasRequiredFields(t *testing.T) {
	pkg := PackageInfo{
		Name:        "neovim",
		Version:     "0.10.0-1",
		Description: "Vim-fork focused on extensibility and usability",
		Source:      SourceOfficial,
	}

	if pkg.Name != "neovim" {
		t.Errorf("Expected Name 'neovim', got '%s'", pkg.Name)
	}
	if pkg.Version != "0.10.0-1" {
		t.Errorf("Expected Version '0.10.0-1', got '%s'", pkg.Version)
	}
	if pkg.Source != SourceOfficial {
		t.Errorf("Expected Source 'official', got '%s'", pkg.Source)
	}
}

func TestPackageSource_Constants(t *testing.T) {
	if SourceOfficial != "official" {
		t.Error("SourceOfficial should be 'official'")
	}
	if SourceAUR != "aur" {
		t.Error("SourceAUR should be 'aur'")
	}
	if SourceShedOS != "shedos" {
		t.Error("SourceShedOS should be 'shedos'")
	}
}

func TestPackageDB_Search_Interface(t *testing.T) {
	// Create a mock that implements PackageDB
	mock := &mockPackageDB{
		packages: []PackageInfo{
			{Name: "neovim", Version: "0.10.0", Source: SourceOfficial},
			{Name: "neovim-nightly", Version: "0.11.0", Source: SourceAUR},
		},
	}

	// Verify it satisfies the interface
	var db PackageDB = mock

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
		packages: []PackageInfo{
			{Name: "neovim", Version: "0.10.0", Source: SourceOfficial},
		},
	}

	var db PackageDB = mock

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
	packages []PackageInfo
}

func (m *mockPackageDB) Search(query string) ([]PackageInfo, error) {
	return m.packages, nil
}

func (m *mockPackageDB) GetInfo(name string) (*PackageInfo, error) {
	for _, p := range m.packages {
		if p.Name == name {
			return &p, nil
		}
	}
	return nil, nil
}
