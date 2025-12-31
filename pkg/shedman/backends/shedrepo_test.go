package backends_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/theshedman/shedman/pkg/shedman/backends"
)

func TestShedRepoBackend_Name(t *testing.T) {
	backend := backends.NewShedRepoBackend()

	if backend.Name() != "shedrepo" {
		t.Errorf("expected name 'shedrepo', got '%s'", backend.Name())
	}
}

func TestShedRepoBackend_Sync_Success(t *testing.T) {
	// Create mock server for both /arch/ and /shed/ endpoints
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		switch r.URL.Path {
		case "/arch/x86_64/shedos.db":
			w.Write([]byte("mock arch database"))
		case "/shed/index.json":
			w.Write([]byte(`{"packages":[]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	backend := backends.NewShedRepoBackendWithURL(server.URL)

	err := backend.Sync()
	if err != nil {
		t.Errorf("Sync should succeed, but got error: %v", err)
	}
}

func TestShedRepoBackend_Sync_ArchFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	backend := backends.NewShedRepoBackendWithURL(server.URL)

	err := backend.Sync()
	if err == nil {
		t.Error("Sync should fail when arch database unavailable")
	}
}

func TestShedRepoBackend_Sync_NetworkError(t *testing.T) {
	backend := backends.NewShedRepoBackendWithURL("http://localhost:99999")

	err := backend.Sync()
	if err == nil {
		t.Error("Sync should fail on network error")
	}
}
