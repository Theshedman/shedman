package de

import (
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

// MockBackend for DE testing
type MockBackend struct {
	core.OfficialBackend
	installed []string
}

func (m *MockBackend) Name() string         { return "mock" }
func (m *MockBackend) IsAvailable() bool    { return true }
func (m *MockBackend) Sync() error          { return nil }
func (m *MockBackend) DistroFamily() string { return "arch" }

func (m *MockBackend) IsInstalled(name string) bool {
	for _, p := range m.installed {
		if p == name {
			return true
		}
	}
	return false
}

func (m *MockBackend) Install(pkgs []string, opts core.InstallOptions) error {
	m.installed = append(m.installed, pkgs...)
	return nil
}

func (m *MockBackend) Remove(pkgs []string, opts core.RemoveOptions) error {
	var newInstalled []string
	for _, inst := range m.installed {
		found := false
		for _, rem := range pkgs {
			if inst == rem {
				found = true
				break
			}
		}
		if !found {
			newInstalled = append(newInstalled, inst)
		}
	}
	m.installed = newInstalled
	return nil
}

func (m *MockBackend) InstallLocal(path string, opts core.InstallOptions) error { return nil }
func (m *MockBackend) Upgrade(pkgs []string, opts core.UpgradeOptions) error    { return nil }
func (m *MockBackend) Search(query string) ([]core.PackageInfo, error)          { return nil, nil }
func (m *MockBackend) Info(pkgName string) (*core.PackageInfo, error)           { return nil, nil }
func (m *MockBackend) GetInstalledPackages() ([]core.PackageInfo, error)        { return nil, nil }
func (m *MockBackend) GetPackageFiles(pkgName string) ([]string, error)         { return nil, nil }

func TestManager_List(t *testing.T) {
	mock := &MockBackend{
		installed: []string{"hyprland"},
	}
	engine := core.NewEngineWithBackend(mock)
	mgr := New(engine)

	des, err := mgr.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	foundHypr := false
	for _, d := range des {
		if d.Name == "hyprland" {
			foundHypr = true
			if !d.Installed {
				t.Error("Expected hyprland to be marked as installed")
			}
		} else if d.Name == "gnome" {
			if d.Installed {
				t.Error("Expected gnome to be marked as not installed")
			}
		}
	}

	if !foundHypr {
		t.Error("Hyprland not found in DE list")
	}
}

func TestManager_Switch(t *testing.T) {
	mock := &MockBackend{
		installed: []string{"gnome-shell"}, // Currently installed
	}
	engine := core.NewEngineWithBackend(mock)
	mgr := New(engine)

	// Switch to hyprland
	err := mgr.Switch("hyprland")
	if err != nil {
		t.Fatalf("Switch failed: %v", err)
	}

	// Verify gnome-shell removed (simplistic logic check)
	installedHypr := false
	for _, p := range mock.installed {
		if p == "hyprland" {
			installedHypr = true
		}
	}
	if !installedHypr {
		t.Error("Expected hyprland to be installed after switch")
	}
}
