package config

import (
	"strings"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

// MockBackend restricts dependencies for testing
type MockBackend struct {
	core.OfficialBackend // Embed to satisfy interface
	pkgs                 []core.PackageInfo
	installed            []string
}

func (m *MockBackend) Name() string         { return "mock" }
func (m *MockBackend) IsAvailable() bool    { return true }
func (m *MockBackend) Sync() error          { return nil }
func (m *MockBackend) DistroFamily() string { return "arch" } // Kept for compatibility if still used, though removed earlier

func (m *MockBackend) Search(query string) ([]core.PackageInfo, error) {
	var results []core.PackageInfo
	for _, p := range m.pkgs {
		if strings.HasPrefix(p.Name, query) || query == "" {
			results = append(results, p)
		}
	}
	return results, nil
}

func (m *MockBackend) IsInstalled(name string) bool {
	for _, p := range m.installed {
		if p == name {
			return true
		}
	}
	return false
}

func (m *MockBackend) InstallLocal(path string, opts core.InstallOptions) error { return nil }
func (m *MockBackend) Install(pkgs []string, opts core.InstallOptions) error {
	m.installed = append(m.installed, pkgs...)
	return nil
}

func (m *MockBackend) Remove(pkgs []string, opts core.RemoveOptions) error   { return nil }
func (m *MockBackend) Upgrade(pkgs []string, opts core.UpgradeOptions) error { return nil }
func (m *MockBackend) Info(pkgName string) (*core.PackageInfo, error)        { return nil, nil }
func (m *MockBackend) GetInstalledPackages() ([]core.PackageInfo, error)     { return nil, nil }
func (m *MockBackend) GetPackageFiles(pkgName string) ([]string, error)      { return nil, nil }

func TestManager_List(t *testing.T) {
	mock := &MockBackend{
		pkgs: []core.PackageInfo{
			{Name: "shedos-configs-hypr", Version: "1.0.0", Description: "Hyprland configs"},
			{Name: "vim", Version: "9.0", Description: "Editor"},
			{Name: "shedos-configs-nvim", Version: "1.0.0", Description: "Neovim configs"},
		},
	}

	engine := core.NewEngineWithBackend(mock)
	mgr := New(engine)

	configs, err := mgr.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(configs) != 2 {
		t.Errorf("Expected 2 config packages, got %d", len(configs))
	}

	expected := map[string]bool{
		"shedos-configs-hypr": true,
		"shedos-configs-nvim": true,
	}

	for _, c := range configs {
		if !expected[c.Name] {
			t.Errorf("Unexpected config package: %s", c.Name)
		}
	}
}

func TestManager_Apply(t *testing.T) {
	mock := &MockBackend{}
	engine := core.NewEngineWithBackend(mock)
	mgr := New(engine)

	err := mgr.Apply("hypr")
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Verify installation
	if len(mock.installed) != 1 {
		t.Fatalf("Expected 1 installed package, got %d", len(mock.installed))
	}

	if mock.installed[0] != "shedos-configs-hypr" {
		t.Errorf("Expected 'shedos-configs-hypr', got '%s'", mock.installed[0])
	}
}
