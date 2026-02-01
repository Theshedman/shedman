package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theshedman/shedman/internal/config"
)

type hookBackend struct {
	installCalled bool
}

func (h *hookBackend) Name() string                                { return "hook-backend" }
func (h *hookBackend) Sync() error                                 { return nil }
func (h *hookBackend) IsAvailable() bool                           { return true }
func (h *hookBackend) Install(pkgs []string, opts InstallOptions) error {
	h.installCalled = true
	return nil
}
func (h *hookBackend) Remove(pkgs []string, opts RemoveOptions) error { return nil }
func (h *hookBackend) IsInstalled(pkgName string) bool                { return false }
func (h *hookBackend) Search(query string) ([]PackageInfo, error)     { return nil, nil }
func (h *hookBackend) Info(pkgName string) (*PackageInfo, error)      { return nil, nil }
func (h *hookBackend) GetInstalledPackages() ([]PackageInfo, error)   { return nil, nil }
func (h *hookBackend) Upgrade(pkgs []string, opts UpgradeOptions) error {
	return nil
}
func (h *hookBackend) InstallLocal(path string, opts InstallOptions) error {
	return nil
}
func (h *hookBackend) GetPackageFiles(pkgName string) ([]string, error) { return nil, nil }
func (h *hookBackend) GetFileOwner(path string) (string, error)         { return "", nil }
func (h *hookBackend) SearchFiles(query string) ([]string, error)       { return nil, nil }
func (h *hookBackend) ListExplicitPackages() ([]string, error)          { return nil, nil }
func (h *hookBackend) Audit() ([]string, error)                         { return nil, nil }
func (h *hookBackend) Diff() ([]PackageDiff, error)                     { return nil, nil }

func TestEngine_InstallHooks(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "hook.log")
	t.Setenv("SHEDMAN_HOOK_LOG", logPath)

	preHook := filepath.Join(tmpDir, "pre.sh")
	postHook := filepath.Join(tmpDir, "post.sh")

	if err := os.WriteFile(preHook, []byte("#!/bin/sh\necho pre >> \"$SHEDMAN_HOOK_LOG\"\n"), 0755); err != nil {
		t.Fatalf("write pre hook: %v", err)
	}
	if err := os.WriteFile(postHook, []byte("#!/bin/sh\necho post >> \"$SHEDMAN_HOOK_LOG\"\n"), 0755); err != nil {
		t.Fatalf("write post hook: %v", err)
	}

	backend := &hookBackend{}
	engine := NewEngineWithBackend(backend)
	engine.SetConfig(&config.Config{
		Hooks: config.HookConfig{
			PreInstall:  preHook,
			PostInstall: postHook,
		},
	})

	if err := engine.Install([]string{"vim"}, InstallOptions{}); err != nil {
		t.Fatalf("Install failed: %v", err)
	}
	if !backend.installCalled {
		t.Error("expected install to be called")
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read hook log: %v", err)
	}
	content := strings.TrimSpace(string(data))
	if content != "pre\npost" && content != "pre\r\npost" {
		t.Errorf("unexpected hook log: %q", content)
	}
}

func TestEngine_PreInstallHookFailure(t *testing.T) {
	tmpDir := t.TempDir()
	preHook := filepath.Join(tmpDir, "pre.sh")
	if err := os.WriteFile(preHook, []byte("#!/bin/sh\nexit 1\n"), 0755); err != nil {
		t.Fatalf("write pre hook: %v", err)
	}

	backend := &hookBackend{}
	engine := NewEngineWithBackend(backend)
	engine.SetConfig(&config.Config{
		Hooks: config.HookConfig{
			PreInstall: preHook,
		},
	})

	if err := engine.Install([]string{"vim"}, InstallOptions{}); err == nil {
		t.Fatal("expected pre-install hook failure")
	}
	if backend.installCalled {
		t.Error("install should not run when pre-install hook fails")
	}
}
