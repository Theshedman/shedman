package boot

import (
	"context"
	"fmt"
	"os/exec"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

// MockBackend for Boot testing
type MockBackend struct {
	core.OfficialBackend
	installed map[string]string // map[name]version
}

func (m *MockBackend) Name() string         { return "mock" }
func (m *MockBackend) IsAvailable() bool    { return true }
func (m *MockBackend) Sync() error          { return nil }
func (m *MockBackend) DistroFamily() string { return "arch" }

func (m *MockBackend) Search(query string) ([]core.PackageInfo, error) {
	return nil, nil
}

func (m *MockBackend) IsInstalled(name string) bool {
	_, ok := m.installed[name]
	return ok
}

func (m *MockBackend) Info(pkgName string) (*core.PackageInfo, error) {
	if ver, ok := m.installed[pkgName]; ok {
		return &core.PackageInfo{
			Name:    pkgName,
			Version: ver,
		}, nil
	}
	return nil, core.ErrPackageNotFound
}

func (m *MockBackend) Install(pkgs []string, opts core.InstallOptions) error    { return nil }
func (m *MockBackend) InstallLocal(path string, opts core.InstallOptions) error { return nil }
func (m *MockBackend) Remove(pkgs []string, opts core.RemoveOptions) error      { return nil }
func (m *MockBackend) Upgrade(pkgs []string, opts core.UpgradeOptions) error    { return nil }
func (m *MockBackend) GetInstalledPackages() ([]core.PackageInfo, error)        { return nil, nil }
func (m *MockBackend) GetPackageFiles(pkgName string) ([]string, error)         { return nil, nil }

// MockExecutor implements Executor interface for testing
type MockExecutor struct {
	LookPathFunc       func(file string) (string, error)
	CombinedOutputFunc func(name string, args ...string) ([]byte, error)
}

func (m *MockExecutor) LookPath(file string) (string, error) {
	if m.LookPathFunc != nil {
		return m.LookPathFunc(file)
	}
	return "", fmt.Errorf("not implemented")
}

// Ensure MockExecutor implements util.Executor
func (m *MockExecutor) Command(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

func (m *MockExecutor) CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}

func (m *MockExecutor) CombinedOutput(name string, args ...string) ([]byte, error) {
	if m.CombinedOutputFunc != nil {
		return m.CombinedOutputFunc(name, args...)
	}
	return nil, fmt.Errorf("not implemented")
}

func TestManager_List(t *testing.T) {
	mock := &MockBackend{
		installed: map[string]string{
			"linux":     "6.6.1-arch1-1",
			"linux-lts": "6.1.60-1",
			"vim":       "9.0",
		},
	}
	engine := core.NewEngineWithBackend(mock)
	mgr := New(engine)

	kernels, err := mgr.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(kernels) != 2 {
		t.Errorf("Expected 2 kernels, got %d", len(kernels))
	}

	foundStock := false
	foundLTS := false

	for _, k := range kernels {
		switch k.Name {
		case "linux":

			foundStock = true
			if k.Version != "6.6.1-arch1-1" {
				t.Errorf("Expected linux version 6.6.1-arch1-1, got %s", k.Version)
			}
		case "linux-lts":
			foundLTS = true
		}

	}

	if !foundStock {
		t.Error("linux kernel not found")
	}
	if !foundLTS {
		t.Error("linux-lts kernel not found")
	}
}

func TestManager_SetDefault(t *testing.T) {
	mock := &MockBackend{
		installed: map[string]string{
			"linux": "6.6.1",
		},
	}
	engine := core.NewEngineWithBackend(mock)
	// Use dummy executor to simulate success
	mockExec := &MockExecutor{
		LookPathFunc: func(file string) (string, error) {
			if file == "bootctl" {
				return "/usr/bin/bootctl", nil
			}
			return "", fmt.Errorf("not found")
		},
		CombinedOutputFunc: func(name string, args ...string) ([]byte, error) {
			return []byte("success"), nil
		},
	}

	mgr := NewWithExecutor(engine, mockExec)

	err := mgr.SetDefault("linux")
	if err != nil {
		t.Errorf("SetDefault(linux) with mock failed: %v", err)
	}

	err = mgr.SetDefault("linux-zen")
	if err == nil {
		t.Error("Expected error when setting default to uninstalled kernel")
	}
}
