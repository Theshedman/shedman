package boot

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
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
func (m *MockBackend) GetFileOwner(path string) (string, error)                 { return "", nil }
func (m *MockBackend) SearchFiles(query string) ([]string, error)               { return nil, nil }
func (m *MockBackend) ListExplicitPackages() ([]string, error)                  { return nil, nil }
func (m *MockBackend) Audit() ([]string, error)                                 { return nil, nil }
func (m *MockBackend) Diff() ([]core.PackageDiff, error)                        { return nil, nil }

// MockExecutor implements Executor interface for testing
type MockExecutor struct {
	LookPathFunc       func(file string) (string, error)
	CombinedOutputFunc func(name string, args ...string) ([]byte, error)
	CombinedCalls      []string
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
	m.CombinedCalls = append(m.CombinedCalls, name+" "+strings.Join(args, " "))
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
	mockExec := &MockExecutor{
		CombinedOutputFunc: func(name string, args ...string) ([]byte, error) {
			if name == "uname" {
				return []byte("6.6.1-arch1-1"), nil
			}
			return []byte(""), nil
		},
	}
	mgr := NewWithExecutor(engine, mockExec)

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
			if !k.Current {
				t.Error("Expected linux to be marked current")
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

func TestManager_SetDefault_GRUB(t *testing.T) {
	mock := &MockBackend{
		installed: map[string]string{
			"linux": "6.6.1",
		},
	}
	engine := core.NewEngineWithBackend(mock)

	grubCfg := "submenu 'Advanced options for Arch Linux' {\n" +
		"  menuentry 'Arch Linux, with Linux linux' {\n" +
		"    linux /boot/vmlinuz-linux root=UUID=123\n" +
		"  }\n" +
		"  menuentry 'Arch Linux, with Linux linux (fallback initramfs)' {\n" +
		"    linux /boot/vmlinuz-linux root=UUID=123\n" +
		"  }\n" +
		"}\n"

	tmp := t.TempDir() + "/grub.cfg"
	if err := os.WriteFile(tmp, []byte(grubCfg), 0644); err != nil {
		t.Fatalf("Failed to write grub cfg: %v", err)
	}

	mockExec := &MockExecutor{
		LookPathFunc: func(file string) (string, error) {
			if file == "bootctl" {
				return "", fmt.Errorf("not found")
			}
			if file == "grub-set-default" {
				return "/usr/bin/grub-set-default", nil
			}
			return "", fmt.Errorf("not found")
		},
		CombinedOutputFunc: func(name string, args ...string) ([]byte, error) {
			return []byte("ok"), nil
		},
	}

	mgr := NewWithExecutor(engine, mockExec)
	mgr.grubConfigPaths = []string{tmp}

	if err := mgr.SetDefault("linux"); err != nil {
		t.Fatalf("SetDefault(linux) via grub failed: %v", err)
	}
	if !callContains(mockExec.CombinedCalls, "grub-set-default Advanced options for Arch Linux>Arch Linux, with Linux linux") {
		t.Errorf("Expected grub-set-default call, got: %v", mockExec.CombinedCalls)
	}
}

func callContains(calls []string, target string) bool {
	for _, call := range calls {
		if call == target {
			return true
		}
	}
	return false
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
