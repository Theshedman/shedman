package shedrepo

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

func TestShedRepoBackend_Name(t *testing.T) {
	c := core.NewFileSystemCacheWithDir(t.TempDir())
	b := New(c, 0)
	if b.Name() != "shedrepo" {
		t.Errorf("Expected name 'shedrepo', got '%s'", b.Name())
	}
}

func TestShedRepoBackend_SetForceRefresh(t *testing.T) {
	c := core.NewFileSystemCacheWithDir(t.TempDir())
	b := New(c, 0)
	b.SetForceRefresh(true)

	if !b.forceRefresh {
		t.Error("Expected forceRefresh to be true")
	}
}

func TestShedRepoBackend_Sync_CacheHit(t *testing.T) {
	c := core.NewFileSystemCacheWithDir(t.TempDir())
	b := New(c, 0)

	// Pre-populate cache so sync skips download
	dbPath := c.GetFilePath("shedrepo", "shedos.db")
	_ = c.WriteFile(dbPath, []byte("cached"))

	indexPath := c.GetFilePath("shedrepo", "index.json")
	_ = c.WriteFile(indexPath, []byte("{}"))

	err := b.Sync()
	if err != nil {
		t.Errorf("Sync failed with cache hit: %v", err)
	}
}

func TestShedRepoBackend_Sync_WithServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test data"))

	}))
	defer server.Close()

	c := core.NewFileSystemCacheWithDir(t.TempDir())
	b := NewWithURL(server.URL, c, 0)
	b.SetForceRefresh(true) // Force download

	err := b.Sync()
	if err != nil {
		t.Errorf("Sync failed: %v", err)
	}
}
