package cache_test

import (
"os"
"path/filepath"
"testing"

"github.com/theshedman/shedman/pkg/shedman/cache"
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
	defer os.RemoveAll(tmpDir)

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
	defer os.RemoveAll(tmpDir)

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
