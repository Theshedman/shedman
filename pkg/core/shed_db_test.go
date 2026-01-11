package core

import (
	"encoding/json"
	"testing"

)

func TestShedDB_NewShedDB(t *testing.T) {
	db := NewShedDB()
	if db == nil {
		t.Fatal("NewShedDB should return non-nil")
	}
}

func TestShedDB_ImplementsPackageDB(t *testing.T) {
	db := NewShedDB()

	// Verify it satisfies the PackageDB interface
	var _ PackageDB = db
}

func TestShedDB_Search(t *testing.T) {
	db := NewShedDB()

	// Mock HTTP client
	db.SetHTTPClient(func(url string) ([]byte, error) {
		response := ShedSearchResponse{
			Packages: []ShedPackage{
				{
					Name:        "shedos-hyprland",
					Version:     "1.0.0-1",
					Description: "ShedOS Hyprland desktop environment",
					Depends:     []string{"hyprland", "waybar", "wofi"},
					Provides:    []string{"shedos-desktop"},
				},
			},
		}
		return json.Marshal(response)
	})

	results, err := db.Search("shedos")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if results[0].Name != "shedos-hyprland" {
		t.Errorf("Expected name 'shedos-hyprland', got '%s'", results[0].Name)
	}

	if results[0].Source != SourceShedOS {
		t.Errorf("Expected source ShedOS, got '%s'", results[0].Source)
	}
}

func TestShedDB_GetInfo(t *testing.T) {
	db := NewShedDB()

	db.SetHTTPClient(func(url string) ([]byte, error) {
		response := ShedInfoResponse{
			Package: &ShedPackage{
				Name:          "shedos-core",
				Version:       "2.0.0-1",
				Description:   "ShedOS core packages",
				Depends:       []string{"base", "linux"},
				OptDepends:    []string{"linux-headers: for kernel module building"},
				Provides:      []string{"shedos-base"},
				Conflicts:     []string{},
				DownloadSize:  10485760, // 10 MiB
				InstalledSize: 52428800, // 50 MiB
				Signature:     "asc",
				Checksum:      "sha256:abc123...",
			},
		}
		return json.Marshal(response)
	})

	info, err := db.GetInfo("shedos-core")
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}

	if info == nil {
		t.Fatal("Expected package info, got nil")
	}

	if info.Name != "shedos-core" {
		t.Errorf("Expected name 'shedos-core', got '%s'", info.Name)
	}
	if info.Version != "2.0.0-1" {
		t.Errorf("Expected version '2.0.0-1', got '%s'", info.Version)
	}
	if len(info.Depends) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(info.Depends))
	}
	if info.Source != SourceShedOS {
		t.Errorf("Expected source ShedOS, got '%s'", info.Source)
	}
	if info.Size != 10485760 {
		t.Errorf("Expected size 10485760, got %d", info.Size)
	}
}

func TestShedDB_GetInfo_NotFound(t *testing.T) {
	db := NewShedDB()

	db.SetHTTPClient(func(url string) ([]byte, error) {
		response := ShedInfoResponse{
			Package: nil,
			Error:   "package not found",
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

func TestShedDB_BuildSearchURL(t *testing.T) {
	db := NewShedDB()
	url := db.BuildSearchURL("shedos")
	expected := "https://repo.shedos.org/api/v1/search?q=shedos"

	if url != expected {
		t.Errorf("Expected URL %q, got %q", expected, url)
	}
}

func TestShedDB_BuildInfoURL(t *testing.T) {
	db := NewShedDB()
	url := db.BuildInfoURL("shedos-core")
	expected := "https://repo.shedos.org/api/v1/package/shedos-core"

	if url != expected {
		t.Errorf("Expected URL %q, got %q", expected, url)
	}
}

func TestShedDB_UsesConfigMirror(t *testing.T) {
	db := NewShedDB()
	// Default config has https://repo.shedos.org
	url := db.BuildSearchURL("test")

	if url[:24] != "https://repo.shedos.org/" {
		t.Errorf("Expected URL to start with default mirror, got %q", url)
	}
}

func TestShedDB_HTTPError(t *testing.T) {
	db := NewShedDB()

	db.SetHTTPClient(func(url string) ([]byte, error) {
		return nil, ErrShedRequestFailed
	})

	_, err := db.Search("test")
	if err == nil {
		t.Error("Expected error for failed HTTP request")
	}
}

func TestShedDB_InvalidJSON(t *testing.T) {
	db := NewShedDB()

	db.SetHTTPClient(func(url string) ([]byte, error) {
		return []byte("invalid json"), nil
	})

	_, err := db.Search("test")
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestShedDB_ShedPackageToPackageInfo(t *testing.T) {
	shedPkg := ShedPackage{
		Name:          "test-pkg",
		Version:       "1.0.0-1",
		Description:   "Test package",
		Depends:       []string{"dep1", "dep2"},
		OptDepends:    []string{"opt1: optional"},
		Provides:      []string{"test"},
		Conflicts:     []string{"test-old"},
		DownloadSize:  1024000,
		InstalledSize: 5120000,
	}

	info := ShedPackageToPackageInfo(shedPkg)

	if info.Name != "test-pkg" {
		t.Errorf("Expected name 'test-pkg', got '%s'", info.Name)
	}
	if info.Version != "1.0.0-1" {
		t.Errorf("Expected version '1.0.0-1', got '%s'", info.Version)
	}
	if len(info.Depends) != 2 {
		t.Errorf("Expected 2 deps, got %d", len(info.Depends))
	}
	if info.Source != SourceShedOS {
		t.Errorf("Expected source ShedOS, got '%s'", info.Source)
	}
	if info.Size != 1024000 {
		t.Errorf("Expected size 1024000, got %d", info.Size)
	}
	if info.InstalledSize != 5120000 {
		t.Errorf("Expected installed size 5120000, got %d", info.InstalledSize)
	}
}
