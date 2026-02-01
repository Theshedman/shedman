package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

type mockSearchBackend struct {
	results []core.PackageInfo
}

func (m *mockSearchBackend) Name() string                                { return "mock" }
func (m *mockSearchBackend) Sync() error                                 { return nil }
func (m *mockSearchBackend) IsAvailable() bool                           { return true }
func (m *mockSearchBackend) Install(pkgs []string, opts core.InstallOptions) error {
	return nil
}
func (m *mockSearchBackend) Remove(pkgs []string, opts core.RemoveOptions) error { return nil }
func (m *mockSearchBackend) IsInstalled(pkgName string) bool                     { return false }
func (m *mockSearchBackend) Search(query string) ([]core.PackageInfo, error) {
	return m.results, nil
}
func (m *mockSearchBackend) Info(pkgName string) (*core.PackageInfo, error)      { return nil, nil }
func (m *mockSearchBackend) GetInstalledPackages() ([]core.PackageInfo, error)   { return nil, nil }
func (m *mockSearchBackend) Upgrade(pkgs []string, opts core.UpgradeOptions) error {
	return nil
}
func (m *mockSearchBackend) InstallLocal(path string, opts core.InstallOptions) error {
	return nil
}
func (m *mockSearchBackend) GetPackageFiles(pkgName string) ([]string, error) { return nil, nil }
func (m *mockSearchBackend) GetFileOwner(path string) (string, error)         { return "", nil }
func (m *mockSearchBackend) SearchFiles(query string) ([]string, error)       { return nil, nil }
func (m *mockSearchBackend) ListExplicitPackages() ([]string, error)          { return nil, nil }
func (m *mockSearchBackend) Audit() ([]string, error)                         { return nil, nil }
func (m *mockSearchBackend) Diff() ([]core.PackageDiff, error)                { return nil, nil }

func TestServer_Health(t *testing.T) {
	engine := core.NewEngine()
	server := NewServer(engine)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() == "" {
		t.Error("expected non-empty health response")
	}
}

func TestServer_Search(t *testing.T) {
	backend := &mockSearchBackend{
		results: []core.PackageInfo{
			{Name: "neovim", Version: "0.9.0"},
		},
	}
	engine := core.NewEngine()
	engine.AddBackend(backend)

	server := NewServer(engine)

	req := httptest.NewRequest(http.MethodGet, "/search?q=neovim", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var out []core.PackageInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(out) != 1 || out[0].Name != "neovim" {
		t.Errorf("unexpected search response: %+v", out)
	}
}

func TestServer_SearchMissingQuery(t *testing.T) {
	engine := core.NewEngine()
	server := NewServer(engine)

	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
