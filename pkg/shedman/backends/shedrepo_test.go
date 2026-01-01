package backends_test

import (
"net/http"
"net/http/httptest"
"os"
"path/filepath"
"testing"

"github.com/theshedman/shedman/pkg/shedman/backends"
"github.com/theshedman/shedman/pkg/shedman/cache"
)

func TestShedRepoBackend_Name(t *testing.T) {
	c := cache.NewFileSystemCache()
	backend := backends.NewShedRepoBackend(c)

	if backend.Name() != "shedrepo" {
		t.Errorf("expected name 'shedrepo', got '%s'", backend.Name())
	}
}

func TestShedRepoBackend_Sync_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
w.WriteHeader(http.StatusOK)
switch r.URL.Path {
case "/arch/x86_64/shedos.db":
w.Write([]byte("mock arch database content"))
case "/shed/index.json":
w.Write([]byte(`{"packages":[]}`))
default:
w.WriteHeader(http.StatusNotFound)
}
}))
	defer server.Close()

	tmpDir := filepath.Join(os.TempDir(), "shedman-shedrepo-test")
	defer os.RemoveAll(tmpDir)

	c := cache.NewFileSystemCacheWithDir(tmpDir)
	backend := backends.NewShedRepoBackendWithURL(server.URL, c)

	err := backend.Sync()
	if err != nil {
		t.Errorf("Sync should succeed, but got error: %v", err)
	}

	// Verify cache files were created
	dbFile := c.GetFilePath("shedrepo", "shedos.db")
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		t.Error("shedos.db should exist after sync")
	}

	indexFile := c.GetFilePath("shedrepo", "index.json")
	if _, err := os.Stat(indexFile); os.IsNotExist(err) {
		t.Error("index.json should exist after sync")
	}
}

func TestShedRepoBackend_Sync_ArchFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
w.WriteHeader(http.StatusInternalServerError)
}))
	defer server.Close()

	tmpDir := filepath.Join(os.TempDir(), "shedman-shedrepo-test")
	defer os.RemoveAll(tmpDir)

	c := cache.NewFileSystemCacheWithDir(tmpDir)
	backend := backends.NewShedRepoBackendWithURL(server.URL, c)

	err := backend.Sync()
	if err == nil {
		t.Error("Sync should fail when arch database unavailable")
	}
}

func TestShedRepoBackend_Sync_NetworkError(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "shedman-shedrepo-test")
	defer os.RemoveAll(tmpDir)

	c := cache.NewFileSystemCacheWithDir(tmpDir)
	backend := backends.NewShedRepoBackendWithURL("http://localhost:99999", c)

	err := backend.Sync()
	if err == nil {
		t.Error("Sync should fail on network error")
	}
}
