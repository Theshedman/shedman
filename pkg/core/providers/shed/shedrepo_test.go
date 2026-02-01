package shedrepo

import (
	"io"
	"net/http"
	"strings"
	"testing"

	shedhttp "github.com/theshedman/shedman/internal/http"
	"github.com/theshedman/shedman/pkg/core"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

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
	client := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("test data")),
			Header:     make(http.Header),
		}, nil
	})}

	c := core.NewFileSystemCacheWithDir(t.TempDir())
	b := NewWithURL("http://mirror1", c, 0)
	b.client = shedhttp.NewRetryClientWithClient([]string{"http://mirror1"}, client)
	b.SetForceRefresh(true) // Force download

	err := b.Sync()
	if err != nil {
		t.Errorf("Sync failed: %v", err)
	}
}
