package shedman

import (
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

type mockBackend struct {
	installCalled bool
	noConfirm     bool
	results       []core.PackageInfo
}

func (m *mockBackend) Name() string                                { return "mock" }
func (m *mockBackend) Sync() error                                 { return nil }
func (m *mockBackend) IsAvailable() bool                           { return true }
func (m *mockBackend) Install(pkgs []string, opts core.InstallOptions) error {
	m.installCalled = true
	m.noConfirm = opts.NoConfirm
	return nil
}
func (m *mockBackend) Remove(pkgs []string, opts core.RemoveOptions) error { return nil }
func (m *mockBackend) IsInstalled(pkgName string) bool                     { return false }
func (m *mockBackend) Search(query string) ([]core.PackageInfo, error) {
	return m.results, nil
}
func (m *mockBackend) Info(pkgName string) (*core.PackageInfo, error)      { return nil, nil }
func (m *mockBackend) GetInstalledPackages() ([]core.PackageInfo, error)   { return nil, nil }
func (m *mockBackend) Upgrade(pkgs []string, opts core.UpgradeOptions) error {
	return nil
}
func (m *mockBackend) InstallLocal(path string, opts core.InstallOptions) error {
	return nil
}
func (m *mockBackend) GetPackageFiles(pkgName string) ([]string, error) { return nil, nil }
func (m *mockBackend) GetFileOwner(path string) (string, error)         { return "", nil }
func (m *mockBackend) SearchFiles(query string) ([]string, error)       { return nil, nil }
func (m *mockBackend) ListExplicitPackages() ([]string, error)          { return nil, nil }
func (m *mockBackend) Audit() ([]string, error)                         { return nil, nil }
func (m *mockBackend) Diff() ([]core.PackageDiff, error)                { return nil, nil }

func TestClient_Search(t *testing.T) {
	backend := &mockBackend{
		results: []core.PackageInfo{{Name: "vim"}},
	}
	engine := core.NewEngine()
	engine.AddBackend(backend)

	client := NewWithEngine(engine)
	results, err := client.Search("vim")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 || results[0].Name != "vim" {
		t.Errorf("unexpected search results: %+v", results)
	}
}

func TestClient_Install_WithConfirmFalse(t *testing.T) {
	backend := &mockBackend{}
	engine := core.NewEngineWithBackend(backend)

	client := NewWithEngine(engine)
	if err := client.Install("vim", WithConfirm(false)); err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	if !backend.installCalled {
		t.Error("expected install to be called")
	}
	if !backend.noConfirm {
		t.Error("expected NoConfirm=true when confirm disabled")
	}
}
