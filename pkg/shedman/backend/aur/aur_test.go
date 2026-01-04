package aur

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/theshedman/shedman/pkg/shedman/cache"
)

func TestAURBackend_Name(t *testing.T) {
	c := cache.NewFileSystemCacheWithDir(t.TempDir())
	b := New(c)
	if b.Name() != "aur" {
		t.Errorf("Expected name 'aur', got '%s'", b.Name())
	}
}

func TestAURBackend_Sync_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"version":5,"type":"multiinfo","resultcount":1,"results":[]}`))
	}))
	defer server.Close()

	c := cache.NewFileSystemCacheWithDir(t.TempDir())
	b := NewWithURL(server.URL+"/", c)
	err := b.Sync()
	if err != nil {
		t.Errorf("Sync failed: %v", err)
	}
}

func TestAURBackend_Sync_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := cache.NewFileSystemCacheWithDir(t.TempDir())
	b := NewWithURL(server.URL+"/", c)
	err := b.Sync()
	if err == nil {
		t.Error("Expected error for 500 response")
	}
}
