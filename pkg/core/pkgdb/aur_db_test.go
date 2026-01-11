package pkgdb_test

import (
	"encoding/json"
	"testing"

	"github.com/theshedman/shedman/pkg/core/pkgdb"
)

func TestAURDB_NewAURDB(t *testing.T) {
	db := pkgdb.NewAURDB()
	if db == nil {
		t.Fatal("NewAURDB should return non-nil")
	}
}

func TestAURDB_ImplementsPackageDB(t *testing.T) {
	db := pkgdb.NewAURDB()

	// Verify it satisfies the PackageDB interface
	var _ pkgdb.PackageDB = db
}

func TestAURDB_Search(t *testing.T) {
	db := pkgdb.NewAURDB()

	// Mock HTTP client
	db.SetHTTPClient(func(url string) ([]byte, error) {
		// Return fake AUR RPC response
		response := pkgdb.AURSearchResponse{
			ResultCount: 1,
			Results: []pkgdb.AURPackage{
				{
					Name:        "neovim-nightly-bin",
					Version:     "0.10.0.r123",
					Description: "Nightly build of Neovim",
					NumVotes:    500,
					Popularity:  10.5,
					Depends:     []string{"glibc", "gcc-libs"},
				},
			},
		}
		return json.Marshal(response)
	})

	results, err := db.Search("neovim-nightly")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if results[0].Name != "neovim-nightly-bin" {
		t.Errorf("Expected name 'neovim-nightly-bin', got '%s'", results[0].Name)
	}

	if results[0].Source != pkgdb.SourceAUR {
		t.Errorf("Expected source AUR, got '%s'", results[0].Source)
	}
}

func TestAURDB_GetInfo(t *testing.T) {
	db := pkgdb.NewAURDB()

	db.SetHTTPClient(func(url string) ([]byte, error) {
		response := pkgdb.AURInfoResponse{
			ResultCount: 1,
			Results: []pkgdb.AURPackage{
				{
					Name:        "yay",
					Version:     "12.3.5-1",
					Description: "Yet another yogurt - An AUR Helper",
					Depends:     []string{"pacman", "git", "go"},
					OptDepends:  []string{"sudo: privilege elevation"},
					Provides:    []string{"aur-helper"},
					Conflicts:   []string{"yay-bin"},
					NumVotes:    5000,
					Popularity:  50.0,
				},
			},
		}
		return json.Marshal(response)
	})

	info, err := db.GetInfo("yay")
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}

	if info == nil {
		t.Fatal("Expected package info, got nil")
	}

	if info.Name != "yay" {
		t.Errorf("Expected name 'yay', got '%s'", info.Name)
	}
	if info.Version != "12.3.5-1" {
		t.Errorf("Expected version '12.3.5-1', got '%s'", info.Version)
	}
	if len(info.Depends) != 3 {
		t.Errorf("Expected 3 dependencies, got %d", len(info.Depends))
	}
	if info.Source != pkgdb.SourceAUR {
		t.Errorf("Expected source AUR, got '%s'", info.Source)
	}
}

func TestAURDB_GetInfo_NotFound(t *testing.T) {
	db := pkgdb.NewAURDB()

	db.SetHTTPClient(func(url string) ([]byte, error) {
		response := pkgdb.AURInfoResponse{
			ResultCount: 0,
			Results:     []pkgdb.AURPackage{},
		}
		return json.Marshal(response)
	})

	info, err := db.GetInfo("nonexistent-package")
	if err != nil {
		t.Fatalf("GetInfo should not error for not found: %v", err)
	}

	if info != nil {
		t.Error("Expected nil for non-existent package")
	}
}

func TestAURDB_BuildAURSearchURL(t *testing.T) {
	db := pkgdb.NewAURDB()
	url := db.BuildSearchURL("neovim")
	expected := "https://aur.archlinux.org/rpc/v5/search/neovim"

	if url != expected {
		t.Errorf("Expected URL %q, got %q", expected, url)
	}
}

func TestAURDB_BuildAURInfoURL(t *testing.T) {
	db := pkgdb.NewAURDB()
	url := db.BuildInfoURL("yay")
	expected := "https://aur.archlinux.org/rpc/v5/info?arg[]=yay"

	if url != expected {
		t.Errorf("Expected URL %q, got %q", expected, url)
	}
}

func TestAURDB_ParseAURPackage(t *testing.T) {
	aurPkg := pkgdb.AURPackage{
		Name:        "test-pkg",
		Version:     "1.0.0-1",
		Description: "Test package",
		Depends:     []string{"dep1", "dep2"},
		OptDepends:  []string{"opt1: optional"},
		Provides:    []string{"test"},
		Conflicts:   []string{"test-old"},
		NumVotes:    100,
		Popularity:  5.5,
	}

	info := pkgdb.AURPackageToPackageInfo(aurPkg)

	if info.Name != "test-pkg" {
		t.Errorf("Expected name 'test-pkg', got '%s'", info.Name)
	}
	if info.Version != "1.0.0-1" {
		t.Errorf("Expected version '1.0.0-1', got '%s'", info.Version)
	}
	if len(info.Depends) != 2 {
		t.Errorf("Expected 2 deps, got %d", len(info.Depends))
	}
	if info.Source != pkgdb.SourceAUR {
		t.Errorf("Expected source AUR, got '%s'", info.Source)
	}
}

func TestAURDB_HTTPError(t *testing.T) {
	db := pkgdb.NewAURDB()

	db.SetHTTPClient(func(url string) ([]byte, error) {
		return nil, pkgdb.ErrAURRequestFailed
	})

	_, err := db.Search("test")
	if err == nil {
		t.Error("Expected error for failed HTTP request")
	}
}

func TestAURDB_InvalidJSON(t *testing.T) {
	db := pkgdb.NewAURDB()

	db.SetHTTPClient(func(url string) ([]byte, error) {
		return []byte("invalid json"), nil
	})

	_, err := db.Search("test")
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}
