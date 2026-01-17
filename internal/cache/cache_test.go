package cache_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/theshedman/shedman/internal/cache"
	"github.com/theshedman/shedman/internal/util"
)

func TestNewFileSystemCache(t *testing.T) {
	c := cache.NewFileSystemCache()

	dir := c.GetDir()
	if dir == "" {
		t.Error("cache dir should not be empty")
	}
	if filepath.Base(dir) != "shedman" {
		t.Errorf("cache dir should end with 'shedman', got: %s", dir)
	}
}

func TestGetSubDir_AUR(t *testing.T) {
	c := cache.NewFileSystemCache()
	dir := c.GetSubDir("aur")

	if !filepath.IsAbs(dir) {
		t.Error("should return absolute path")
	}
	if filepath.Base(dir) != "aur" {
		t.Errorf("should end with 'aur', got: %s", filepath.Base(dir))
	}
}

func TestGetSubDir_ShedRepo(t *testing.T) {
	c := cache.NewFileSystemCache()
	dir := c.GetSubDir("shedrepo")

	if filepath.Base(dir) != "shedrepo" {
		t.Errorf("should end with 'shedrepo', got: %s", filepath.Base(dir))
	}
}

func TestGetFilePath_AURPackages(t *testing.T) {
	c := cache.NewFileSystemCache()
	path := c.GetFilePath("aur", "packages.json")

	if filepath.Base(path) != "packages.json" {
		t.Errorf("should end with 'packages.json', got: %s", filepath.Base(path))
	}
	if filepath.Base(filepath.Dir(path)) != "aur" {
		t.Errorf("parent dir should be 'aur', got: %s", filepath.Base(filepath.Dir(path)))
	}
}

func TestGetFilePath_ShedOSDB(t *testing.T) {
	c := cache.NewFileSystemCache()
	path := c.GetFilePath("shedrepo", "shedos.db")

	if filepath.Base(path) != "shedos.db" {
		t.Errorf("should end with 'shedos.db', got: %s", filepath.Base(path))
	}
}

func TestGetFilePath_ShedIndex(t *testing.T) {
	c := cache.NewFileSystemCache()
	path := c.GetFilePath("shedrepo", "index.json")

	if filepath.Base(path) != "index.json" {
		t.Errorf("should end with 'index.json', got: %s", filepath.Base(path))
	}
}

func TestEnsureDir(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "shedman-test-cache")
	defer func() { _ = os.RemoveAll(tmpDir) }()

	c := cache.NewFileSystemCacheWithDir(tmpDir)
	testDir := c.GetSubDir("testdir")

	err := c.EnsureDir(testDir)
	if err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	info, err := os.Stat(testDir)
	if err != nil {
		t.Fatalf("directory should exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("should be a directory")
	}
}

func TestWriteAndReadFile(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "shedman-test-cache")
	defer func() { _ = os.RemoveAll(tmpDir) }()

	c := cache.NewFileSystemCacheWithDir(tmpDir)
	testFile := c.GetFilePath("aur", "test.json")
	testData := []byte(`{"test": true}`)

	// Write
	err := c.WriteFile(testFile, testData)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Read
	data, err := c.ReadFile(testFile)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if string(data) != string(testData) {
		t.Errorf("data mismatch: expected %s, got %s", testData, data)
	}
}

func TestCache_IsFresh_ReturnsTrue_WhenFileIsRecent(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "shedman-test-fresh")
	defer func() { _ = os.RemoveAll(tmpDir) }()

	c := cache.NewFileSystemCacheWithDir(tmpDir)
	testFile := c.GetFilePath("test", "recent.json")

	// Write a file (will have current timestamp)
	err := c.WriteFile(testFile, []byte("test"))
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// File written just now should be fresh for 1 hour
	if !c.IsFresh(testFile, 1*time.Hour) {
		t.Error("file written just now should be fresh")
	}
}

func TestCache_IsFresh_ReturnsFalse_WhenFileIsStale(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "shedman-test-stale")
	defer func() { _ = os.RemoveAll(tmpDir) }()

	c := cache.NewFileSystemCacheWithDir(tmpDir)
	testFile := c.GetFilePath("test", "stale.json")

	// Write a file
	err := c.WriteFile(testFile, []byte("test"))
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// File is stale if maxAge is 0 (always stale)
	if c.IsFresh(testFile, 0) {
		t.Error("file should be stale when maxAge is 0")
	}
}

func TestCache_IsFresh_ReturnsFalse_WhenFileNotExists(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "shedman-test-noexist")
	defer func() { _ = os.RemoveAll(tmpDir) }()

	c := cache.NewFileSystemCacheWithDir(tmpDir)
	testFile := c.GetFilePath("test", "nonexistent.json")

	if c.IsFresh(testFile, 1*time.Hour) {
		t.Error("non-existent file should not be fresh")
	}
}

func TestCache_GetModTime_ReturnsTime(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "shedman-test-modtime")
	defer func() { _ = os.RemoveAll(tmpDir) }()

	c := cache.NewFileSystemCacheWithDir(tmpDir)
	testFile := c.GetFilePath("test", "modtime.json")

	before := time.Now().Add(-1 * time.Second)
	err := c.WriteFile(testFile, []byte("test"))
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	after := time.Now().Add(1 * time.Second)

	modTime, err := c.GetModTime(testFile)
	if err != nil {
		t.Fatalf("GetModTime failed: %v", err)
	}

	if modTime.Before(before) || modTime.After(after) {
		t.Errorf("modTime %v should be between %v and %v", modTime, before, after)
	}
}

func TestCache_GetModTime_ErrorsWhenNotExists(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "shedman-test-modtime-err")
	defer func() { _ = os.RemoveAll(tmpDir) }()

	c := cache.NewFileSystemCacheWithDir(tmpDir)
	testFile := c.GetFilePath("test", "nonexistent.json")

	_, err := c.GetModTime(testFile)
	if err == nil {
		t.Error("GetModTime should error for non-existent file")
	}
}

func TestFileSystemCache_FindVersions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "shedman-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	c := cache.NewFileSystemCacheWithDir(tmpDir)

	// Setup dummy package files
	// neovim-0.9.0-1-x86_64.pkg.tar.zst
	// neovim-0.8.0-1-x86_64.pkg.tar.zst
	// other-1.0.0-1-x86_64.pkg.tar.zst
	files := []string{
		"neovim-0.9.0-1-x86_64.pkg.tar.zst",
		"neovim-0.8.0-1-x86_64.pkg.tar.zst",
		"other-1.0.0-1-x86_64.pkg.tar.zst",
	}

	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if err := os.WriteFile(path, []byte("dummy"), util.FilePermissions); err != nil {
			t.Fatalf("Failed to create dummy file %s: %v", f, err)
		}
	}

	// Test finding versions
	matches, err := c.FindVersions(tmpDir, "neovim")
	if err != nil {
		t.Fatalf("FindVersions failed: %v", err)
	}

	if len(matches) != 2 {
		t.Errorf("Expected 2 matches, got %d", len(matches))
	}

	// Verify content
	found090 := false
	found080 := false

	for _, m := range matches {
		if m.Name != "neovim" {
			t.Errorf("Expected name neovim, got %s", m.Name)
		}
		switch m.Version {
		case "0.9.0-1":

			found090 = true
		case "0.8.0-1":

			found080 = true
		}
	}

	if !found090 {
		t.Error("Did not find version 0.9.0-1")
	}
	if !found080 {
		t.Error("Did not find version 0.8.0-1")
	}
}
