package backends_test

import (
"net/http"
"net/http/httptest"
"testing"

"github.com/theshedman/shedman/pkg/shedman/backends"
)

func TestAURBackend_Name(t *testing.T) {
	backend := backends.NewAURBackend()

	if backend.Name() != "aur" {
		t.Errorf("expected name 'aur', got '%s'", backend.Name())
	}
}

func TestAURBackend_Sync_Success(t *testing.T) {
	// Create a mock server that returns a valid AUR response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"version":5,"type":"info","resultcount":1,"results":[]}`))
	}))
	defer server.Close()

	// Create backend with mock server URL
	backend := backends.NewAURBackendWithURL(server.URL + "/")

	err := backend.Sync()
	if err != nil {
		t.Errorf("Sync should succeed, but got error: %v", err)
	}
}

func TestAURBackend_Sync_ServerError(t *testing.T) {
	// Create a mock server that returns 500 Internal Server Error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
w.WriteHeader(http.StatusInternalServerError)
}))
	defer server.Close()

	backend := backends.NewAURBackendWithURL(server.URL + "/")

	err := backend.Sync()
	if err == nil {
		t.Error("Sync should fail on server error, but got nil")
	}
}

func TestAURBackend_Sync_APIError(t *testing.T) {
	// Create a mock server that returns an AUR API error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"version":5,"type":"error","resultcount":0,"results":[],"error":"Invalid request"}`))
	}))
	defer server.Close()

	backend := backends.NewAURBackendWithURL(server.URL + "/")

	err := backend.Sync()
	if err == nil {
		t.Error("Sync should fail on API error, but got nil")
	}
}

func TestAURBackend_Sync_InvalidJSON(t *testing.T) {
	// Create a mock server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json}`))
	}))
	defer server.Close()

	backend := backends.NewAURBackendWithURL(server.URL + "/")

	err := backend.Sync()
	if err == nil {
		t.Error("Sync should fail on invalid JSON, but got nil")
	}
}

func TestAURBackend_Sync_NetworkError(t *testing.T) {
	// Use an invalid URL to simulate network error
	backend := backends.NewAURBackendWithURL("http://localhost:99999/")

	err := backend.Sync()
	if err == nil {
		t.Error("Sync should fail on network error, but got nil")
	}
}
