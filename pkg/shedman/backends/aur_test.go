package backends_test

import (
"net/http"
"net/http/httptest"
"testing"

"github.com/theshedman/shedman/pkg/shedman/backends"
"github.com/theshedman/shedman/pkg/shedman/cache"
)

func TestAURBackend_Name(t *testing.T) {
	c := cache.NewFileSystemCache()
	backend := backends.NewAURBackend(c)

	if backend.Name() != "aur" {
		t.Errorf("expected name 'aur', got '%s'", backend.Name())
	}
}

func TestAURBackend_Sync_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"version":5,"type":"info","resultcount":1,"results":[{"Name":"linux"}]}`))
	}))
	defer server.Close()

	c := cache.NewFileSystemCache()
	backend := backends.NewAURBackendWithURL(server.URL+"/", c)

	err := backend.Sync()
	if err != nil {
		t.Errorf("Sync should succeed, but got error: %v", err)
	}
}

func TestAURBackend_Sync_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
w.WriteHeader(http.StatusInternalServerError)
}))
	defer server.Close()

	c := cache.NewFileSystemCache()
	backend := backends.NewAURBackendWithURL(server.URL+"/", c)

	err := backend.Sync()
	if err == nil {
		t.Error("Sync should fail on server error, but got nil")
	}
}

func TestAURBackend_Sync_NetworkError(t *testing.T) {
	c := cache.NewFileSystemCache()
	backend := backends.NewAURBackendWithURL("http://localhost:99999/", c)

	err := backend.Sync()
	if err == nil {
		t.Error("Sync should fail on network error, but got nil")
	}
}
