package de

import (
	"testing"

	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/snapshot"
)

// MockSnapshotManager
type MockSnapshotManager struct {
	snapshot.Manager
	createdSnapshot bool
}

func (m *MockSnapshotManager) Create(desc string, opts snapshot.CreateOptions) (*snapshot.Snapshot, error) {
	m.createdSnapshot = true
	return &snapshot.Snapshot{ID: "test-snap-id"}, nil
}

// MockServiceManager
type MockServiceManager struct {
	enabled  string
	disabled string
}

func (m *MockServiceManager) Enable(service string) error {
	m.enabled = service
	return nil
}
func (m *MockServiceManager) Disable(service string) error {
	m.disabled = service
	return nil
}
func (m *MockServiceManager) Start(service string) error   { return nil }
func (m *MockServiceManager) Stop(service string) error    { return nil }
func (m *MockServiceManager) Restart(service string) error { return nil }
func (m *MockServiceManager) IsActive(service string) (bool, error) {
	return false, nil
}
func (m *MockServiceManager) IsEnabled(service string) (bool, error) {
	return false, nil
}

// MockBackend for DE testing
type MockBackend struct {
	core.OfficialBackend
	installed []string
	removed   []string
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
	m.removed = append(m.removed, pkgs...)
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

// Stubs for interface satisfaction
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
	// New needs updated signature or setter injection
	mgr := New(engine)

	des, err := mgr.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	foundHypr := false
	for _, d := range des {
		if d.ID == "hyprland" {
			foundHypr = true
			if !d.Installed {
				t.Error("Expected hyprland to be marked as installed")
			}
		}
	}

	if !foundHypr {
		t.Error("Hyprland not found in DE list")
	}
}

// MockConfigApplier
type MockConfigApplier struct {
	applied []string
}

func (m *MockConfigApplier) Apply(path string) error {
	m.applied = append(m.applied, path)
	return nil
}

func TestManager_Switch(t *testing.T) {
	// Setup Mock Backend
	mockBackend := &MockBackend{
		installed: []string{"gnome"},
	}
	engine := core.NewEngineWithBackend(mockBackend)

	// Setup Mocks
	mockSnap := &MockSnapshotManager{}
	mockSvc := &MockServiceManager{}
	mockApplier := &MockConfigApplier{}

	// Setup Manager
	mgr := New(engine)
	mgr.SetSnapshotManager(mockSnap)
	mgr.SetServiceManager(mockSvc)
	mgr.SetConfigApplier(mockApplier)

	// Inject Test Config into Group Registry for this test
	// We modify the global state carefully
	hyprGroup := core.DefaultGroups["shedos-hyprland"]
	originalConfigs := hyprGroup.Configs
	hyprGroup.Configs = []string{"/etc/hypr/hyprland.conf"}
	core.DefaultGroups["shedos-hyprland"] = hyprGroup
	defer func() {
		// Restore
		hyprGroup.Configs = originalConfigs
		core.DefaultGroups["shedos-hyprland"] = hyprGroup
	}()

	// Options
	opts := SwitchOptions{
		NoSnapshot: false,
		KeepOld:    false,
		NoConfirm:  true,
		DryRun:     false,
	}

	// Switch to hyprland
	err := mgr.Switch("hyprland", opts)
	if err != nil {
		t.Fatalf("Switch failed: %v", err)
	}

	// Verify Snapshot Created
	if !mockSnap.createdSnapshot {
		t.Error("Expected snapshot to be created")
	}

	// Verify Old DE Removed
	expectedRemoved := []string{"gnome", "gnome-circle"}
	for _, expected := range expectedRemoved {
		found := false
		for _, p := range mockBackend.removed {
			if p == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected package %s to be removed", expected)
		}
	}

	// Verify Service Enabled
	if mockSvc.enabled != "sddm" {
		t.Errorf("Expected service sddm to be enabled, got %s", mockSvc.enabled)
	}

	// Verify Old Service Disabled
	if mockSvc.disabled != "gdm" {
		t.Errorf("Expected service gdm to be disabled, got %s", mockSvc.disabled)
	}

	// Verify Config Applied
	if len(mockApplier.applied) != 1 || mockApplier.applied[0] != "/etc/hypr/hyprland.conf" {
		t.Errorf("Expected config /etc/hypr/hyprland.conf to be applied, got %v", mockApplier.applied)
	}
}
