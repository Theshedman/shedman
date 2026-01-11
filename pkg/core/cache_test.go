package core

import (
	"testing"
	"time"

	"github.com/theshedman/shedman/internal/config"
)

func TestPackageFileCache_New(t *testing.T) {
	cache := NewPackageFileCache(5 * time.Minute)
	if cache == nil {
		t.Fatal("Expected non-nil cache")
	}
}

func TestPackageFileCache_DefaultCache(t *testing.T) {
	cache := DefaultPackageFileCache()
	if cache == nil {
		t.Fatal("Expected non-nil cache")
	}
}

func TestPackageFileCache_IsValid_Empty(t *testing.T) {
	cache := NewPackageFileCache(5 * time.Minute)
	if cache.IsValid() {
		t.Error("Empty cache should not be valid")
	}
}

func TestPackageFileCache_SetAndGet(t *testing.T) {
	cache := NewPackageFileCache(1 * time.Hour)

	files := []string{"/usr/bin/test", "/usr/lib/test.so"}
	cache.SetPackageFiles("testpkg", files)

	retrieved, valid := cache.GetPackageFiles("testpkg")
	if !valid {
		t.Error("Cache should be valid after set")
	}
	if len(retrieved) != 2 {
		t.Errorf("Expected 2 files, got %d", len(retrieved))
	}
}

func TestPackageFileCache_GetFileOwner(t *testing.T) {
	cache := NewPackageFileCache(1 * time.Hour)

	files := []string{"/usr/bin/myapp"}
	cache.SetPackageFiles("mypackage", files)

	owner, valid := cache.GetFileOwner("/usr/bin/myapp")
	if !valid {
		t.Error("Cache should be valid")
	}
	if owner != "mypackage" {
		t.Errorf("Expected owner 'mypackage', got '%s'", owner)
	}
}

func TestPackageFileCache_Invalidate(t *testing.T) {
	cache := NewPackageFileCache(1 * time.Hour)
	cache.SetPackageFiles("pkg", []string{"/test"})

	cache.Invalidate()

	if cache.IsValid() {
		t.Error("Cache should be invalid after Invalidate()")
	}
}

func TestPackageFileCache_CheckConflicts(t *testing.T) {
	cache := NewPackageFileCache(1 * time.Hour)
	cache.SetPackageFiles("existingpkg", []string{"/usr/bin/shared"})

	newFiles := []string{"/usr/bin/shared", "/usr/bin/newfile"}
	conflicts := cache.CheckConflicts(newFiles, "newpkg")

	if len(conflicts) != 1 {
		t.Errorf("Expected 1 conflict, got %d", len(conflicts))
	}
	if len(conflicts) > 0 && conflicts[0].FilePath != "/usr/bin/shared" {
		t.Errorf("Expected conflict at '/usr/bin/shared'")
	}
}

func TestOptionsFromConfig(t *testing.T) {
	cfg := config.Default()
	cfg.Network.ParallelDownloads = 10
	cfg.General.Confirm = false

	opts := OptionsFromConfig(cfg)

	if opts.ParallelDownloads != 10 {
		t.Errorf("Expected ParallelDownloads=10, got %d", opts.ParallelDownloads)
	}
	if !opts.NoConfirm {
		t.Error("Expected NoConfirm=true when Confirm=false")
	}
}

func TestOptions_FluentAPI(t *testing.T) {
	opts := DefaultOptions().
		WithNeeded().
		WithAsDeps().
		WithNoConfirm()

	if !opts.Needed {
		t.Error("Expected Needed=true")
	}
	if !opts.AsDeps {
		t.Error("Expected AsDeps=true")
	}
	if opts.AsExplicit {
		t.Error("Expected AsExplicit=false after WithAsDeps")
	}
	if !opts.NoConfirm {
		t.Error("Expected NoConfirm=true")
	}
}

func TestFileConflictType_String(t *testing.T) {
	if FileConflictOwnership.String() != "owned" {
		t.Error("Expected 'owned'")
	}
	if FileConflictExisting.String() != "existing" {
		t.Error("Expected 'existing'")
	}
}
