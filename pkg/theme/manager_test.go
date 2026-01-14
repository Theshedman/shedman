package theme

import (
	"strings"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

// MockBackend for Theme testing
type MockBackend struct {
	core.OfficialBackend
	pkgs      []core.PackageInfo
	installed []string
}

func (m *MockBackend) Name() string         { return "mock" }
func (m *MockBackend) IsAvailable() bool    { return true }
func (m *MockBackend) Sync() error          { return nil }
func (m *MockBackend) DistroFamily() string { return "arch" }

func (m *MockBackend) Search(query string) ([]core.PackageInfo, error) {
	var results []core.PackageInfo
	for _, p := range m.pkgs {
		if strings.HasPrefix(p.Name, query) {
			results = append(results, p)
		}
	}
	return results, nil
}

func (m *MockBackend) Install(pkgs []string, opts core.InstallOptions) error {
	m.installed = append(m.installed, pkgs...)
	return nil
}

func (m *MockBackend) IsInstalled(name string) bool                             { return false }
func (m *MockBackend) InstallLocal(path string, opts core.InstallOptions) error { return nil }
func (m *MockBackend) Remove(pkgs []string, opts core.RemoveOptions) error      { return nil }
func (m *MockBackend) Upgrade(pkgs []string, opts core.UpgradeOptions) error    { return nil }
func (m *MockBackend) Info(pkgName string) (*core.PackageInfo, error)           { return nil, nil }
func (m *MockBackend) GetInstalledPackages() ([]core.PackageInfo, error)        { return nil, nil }
func (m *MockBackend) GetPackageFiles(pkgName string) ([]string, error)         { return nil, nil }

func TestManager_List(t *testing.T) {
	mock := &MockBackend{
		pkgs: []core.PackageInfo{
			{Name: "shedos-theme-catppuccin", Version: "1.0"},
			{Name: "shedos-theme-dracula", Version: "1.0"},
			{Name: "vim", Version: "9.0"},
		},
	}
	engine := core.NewEngineWithBackend(mock)
	mgr := New(engine)

	themes, err := mgr.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(themes) != 2 {
		t.Errorf("Expected 2 themes, got %d", len(themes))
	}

	expected := map[string]bool{
		"shedos-theme-catppuccin": true,
		"shedos-theme-dracula":    true,
	}

	for _, th := range themes {
		if !expected[th.Name] {
			t.Errorf("Unexpected theme: %s", th.Name)
		}
	}
}

func TestManager_Apply(t *testing.T) {
	mock := &MockBackend{}
	engine := core.NewEngineWithBackend(mock)
	mgr := New(engine)

	// Apply "catppuccin" -> should install "shedos-theme-catppuccin"
	err := mgr.Apply("catppuccin")
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	if len(mock.installed) != 1 {
		t.Fatalf("Expected 1 installed package, got %d", len(mock.installed))
	}

	if mock.installed[0] != "shedos-theme-catppuccin" {
		t.Errorf("Expected 'shedos-theme-catppuccin', got '%s'", mock.installed[0])
	}
}
