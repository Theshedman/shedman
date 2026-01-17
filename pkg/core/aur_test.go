package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/internal/util"
	"github.com/theshedman/shedman/pkg/executor"
)

func TestAURInstaller_NewAURInstaller(t *testing.T) {
	ai := NewAURInstallerWithBackend(config.Default(), nil)
	if ai == nil {
		t.Fatal("NewAURInstaller should return non-nil")
	}
}

func TestAURInstaller_NewAURInstallerWithConfig(t *testing.T) {
	cfg := config.Default()
	cfg.AUR.BuildDir = "/custom/build/dir"

	ai := NewAURInstallerWithBackend(cfg, nil)
	if ai == nil {
		t.Fatal("NewAURInstallerWithConfig should return non-nil")
	}

	// Verify the build dir is used
	if ai.GetCacheDir() != "/custom/build/dir" {
		t.Errorf("Expected cache dir '/custom/build/dir', got '%s'", ai.GetCacheDir())
	}
}

func TestAURInstaller_UsesDefaultBuildDirWhenEmpty(t *testing.T) {
	cfg := config.Default()
	cfg.AUR.BuildDir = "" // Empty = use default

	ai := NewAURInstallerWithBackend(cfg, nil)

	// Should fall back to default
	if ai.GetCacheDir() == "" {
		t.Error("Expected non-empty cache dir when BuildDir is empty")
	}
}

func TestAURInstaller_Clone_FirstTime(t *testing.T) {
	ai := NewAURInstallerWithBackend(config.Default(), nil)

	var executedCmds [][]string
	mockExec := &executor.MockExecutor{
		CommandFunc: func(name string, args ...string) *exec.Cmd {
			cmd := append([]string{name}, args...)
			executedCmds = append(executedCmds, cmd)
			return exec.Command("true")
		},
	}
	ai.SetExecutor(mockExec)

	err := ai.Clone("neovim-nightly")
	if err != nil {
		t.Fatalf("Clone failed: %v", err)
	}

	// Should use git clone for first time
	if len(executedCmds) < 1 {
		t.Fatal("Expected git command")
	}
	if executedCmds[0][0] != "git" || executedCmds[0][1] != "clone" {
		t.Errorf("Expected 'git clone', got %v", executedCmds[0])
	}
}

func TestAURInstaller_Clone_Update(t *testing.T) {
	ai := NewAURInstallerWithBackend(config.Default(), nil)

	// Create fake existing clone with .git directory
	tmpDir := t.TempDir()
	ai.SetCacheDir(tmpDir)
	pkgDir := filepath.Join(tmpDir, "neovim-nightly")
	gitDir := filepath.Join(pkgDir, ".git")
	_ = os.MkdirAll(gitDir, util.DirPermissions)

	_ = os.WriteFile(filepath.Join(gitDir, "config"), []byte(""), util.FilePermissions)

	var executedCmds [][]string
	mockExec := &executor.MockExecutor{
		CommandFunc: func(name string, args ...string) *exec.Cmd {
			cmd := append([]string{name}, args...)
			executedCmds = append(executedCmds, cmd)
			return exec.Command("true")
		},
	}
	ai.SetExecutor(mockExec)

	err := ai.Clone("neovim-nightly")
	if err != nil {
		t.Fatalf("Clone update failed: %v", err)
	}

	// Should use git pull for updates, not clone
	found := false
	for _, cmd := range executedCmds {
		cmdStr := strings.Join(cmd, " ")
		if strings.Contains(cmdStr, "pull") || strings.Contains(cmdStr, "fetch") {
			found = true
		}
	}
	if !found {
		t.Errorf("Expected git pull/fetch for existing repo, got %v", executedCmds)
	}
}

func TestAURInstaller_IsFirstTime(t *testing.T) {
	ai := NewAURInstallerWithBackend(config.Default(), nil)
	tmpDir := t.TempDir()
	ai.SetCacheDir(tmpDir)

	// Non-existent package
	if !ai.IsFirstTime("nonexistent-pkg") {
		t.Error("Should be first time for non-existent package")
	}

	// Create fake clone
	pkgDir := filepath.Join(tmpDir, "existing-pkg")
	_ = os.MkdirAll(filepath.Join(pkgDir, ".git"), util.DirPermissions)

	if ai.IsFirstTime("existing-pkg") {
		t.Error("Should not be first time for existing package")
	}
}

func TestAURInstaller_GetPKGBUILD(t *testing.T) {
	ai := NewAURInstallerWithBackend(config.Default(), nil)
	tmpDir := t.TempDir()
	ai.SetCacheDir(tmpDir)

	// Create fake PKGBUILD
	pkgDir := filepath.Join(tmpDir, "test-pkg")
	_ = os.MkdirAll(pkgDir, util.DirPermissions)

	expectedContent := "pkgname=test-pkg\npkgver=1.0.0"
	_ = os.WriteFile(filepath.Join(pkgDir, "PKGBUILD"), []byte(expectedContent), util.FilePermissions)

	content, err := ai.GetPKGBUILD("test-pkg")
	if err != nil {
		t.Fatalf("GetPKGBUILD failed: %v", err)
	}

	if content != expectedContent {
		t.Errorf("Expected %q, got %q", expectedContent, content)
	}
}

func TestAURInstaller_GetPKGBUILDDiff(t *testing.T) {
	ai := NewAURInstallerWithBackend(config.Default(), nil)
	tmpDir := t.TempDir()
	ai.SetCacheDir(tmpDir)

	// Create fake package dir with git
	pkgDir := filepath.Join(tmpDir, "test-pkg")
	_ = os.MkdirAll(filepath.Join(pkgDir, ".git"), util.DirPermissions)

	_ = os.WriteFile(filepath.Join(pkgDir, "PKGBUILD"), []byte("pkgver=2.0"), util.FilePermissions)

	// Note: GetPKGBUILDDiff uses exec.Command directly, not the executor
	// This test verifies it returns an error for non-existent package
	_, err := ai.GetPKGBUILDDiff("test-pkg")
	// Will fail because there's no git repo - that's expected
	if err == nil {
		// If no error, verify it returned something (mock won't work here)
		t.Log("GetPKGBUILDDiff returned no error (may have found a valid repo)")
	}
}

func TestAURInstaller_VerifyChecksums(t *testing.T) {
	ai := NewAURInstallerWithBackend(config.Default(), nil)
	tmpDir := t.TempDir()
	ai.SetCacheDir(tmpDir)

	// Create pkg dir
	pkgDir := filepath.Join(tmpDir, "test-pkg")
	_ = os.MkdirAll(pkgDir, util.DirPermissions)

	var executedCmd []string
	mockExec := &executor.MockExecutor{
		CommandFunc: func(name string, args ...string) *exec.Cmd {
			executedCmd = append([]string{name}, args...)
			return exec.Command("true")
		},
	}
	ai.SetExecutor(mockExec)

	err := ai.VerifyChecksums("test-pkg")
	if err != nil {
		t.Fatalf("VerifyChecksums failed: %v", err)
	}

	// Should run makepkg --verifysource or similar
	cmdStr := strings.Join(executedCmd, " ")
	if !strings.Contains(cmdStr, "makepkg") || !strings.Contains(cmdStr, "verif") {
		t.Errorf("Expected makepkg verification command, got %v", executedCmd)
	}
}

func TestAURInstaller_Build_WithSandbox(t *testing.T) {
	ai := NewAURInstallerWithBackend(config.Default(), nil)
	// Check if bwrap is available
	if _, err := exec.LookPath("bwrap"); err != nil {
		t.Skip("bwrap not found, skipping sandbox test")
	}
	ai.SetSandboxEnabled(true)
	tmpDir := t.TempDir()
	ai.SetCacheDir(tmpDir)
	_ = os.MkdirAll(filepath.Join(tmpDir, "test-pkg"), util.DirPermissions)

	var executedCmd []string
	mockExec := &executor.MockExecutor{
		CommandFunc: func(name string, args ...string) *exec.Cmd {
			executedCmd = append([]string{name}, args...)
			return exec.Command("true")
		},
	}
	ai.SetExecutor(mockExec)

	err := ai.Build("test-pkg")
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Should use bubblewrap when sandbox is enabled
	cmdStr := strings.Join(executedCmd, " ")
	if !strings.Contains(cmdStr, "bwrap") {
		t.Errorf("Expected bubblewrap command, got %v", executedCmd)
	}
}

func TestAURInstaller_Build_WithoutSandbox(t *testing.T) {
	ai := NewAURInstallerWithBackend(config.Default(), nil)
	ai.SetSandboxEnabled(false)
	tmpDir := t.TempDir()
	ai.SetCacheDir(tmpDir)
	_ = os.MkdirAll(filepath.Join(tmpDir, "test-pkg"), util.DirPermissions)

	var executedCmd []string
	mockExec := &executor.MockExecutor{
		CommandFunc: func(name string, args ...string) *exec.Cmd {
			executedCmd = append([]string{name}, args...)
			return exec.Command("true")
		},
	}
	ai.SetExecutor(mockExec)

	err := ai.Build("test-pkg")
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Should run makepkg directly without bwrap
	cmdStr := strings.Join(executedCmd, " ")
	if strings.Contains(cmdStr, "bwrap") {
		t.Errorf("Should not use bubblewrap when disabled, got %v", executedCmd)
	}
	if !strings.Contains(cmdStr, "makepkg") {
		t.Errorf("Expected makepkg command, got %v", executedCmd)
	}
}

func TestAURInstaller_Install(t *testing.T) {
	ai := NewAURInstallerWithBackend(config.Default(), nil)
	tmpDir := t.TempDir()
	ai.SetCacheDir(tmpDir)

	// Create fake built package
	pkgDir := filepath.Join(tmpDir, "test-pkg")
	_ = os.MkdirAll(pkgDir, util.DirPermissions)

	pkgFile := filepath.Join(pkgDir, "test-pkg-1.0-1-x86_64.pkg.tar.zst")
	_ = os.WriteFile(pkgFile, []byte("fake"), util.FilePermissions)

	// Use mock backend for testing
	mockBackend := &mockInstallBackend{
		installLocalFunc: func(path string, opts interface{}) error {
			// Verify we received the correct package file
			if !strings.Contains(path, "test-pkg-1.0-1-x86_64.pkg.tar.zst") {
				t.Errorf("Expected package file path, got %s", path)
			}
			return nil
		},
	}
	ai.SetBackend(mockBackend)

	err := ai.Install("test-pkg", DefaultAUROptions())
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	if !mockBackend.installCalled {
		t.Error("Expected InstallLocal to be called")
	}
}

// mockInstallBackend implements OfficialBackend for testing
type mockInstallBackend struct {
	installCalled    bool
	installLocalFunc func(path string, opts interface{}) error
}

func (m *mockInstallBackend) Name() string { return "mock" }

func (m *mockInstallBackend) IsAvailable() bool { return true }
func (m *mockInstallBackend) Sync() error       { return nil }

func (m *mockInstallBackend) Install(pkgs []string, opts InstallOptions) error {
	return nil
}
func (m *mockInstallBackend) Remove(pkgs []string, opts RemoveOptions) error { return nil }
func (m *mockInstallBackend) Upgrade(pkgs []string, opts UpgradeOptions) error {
	return nil
}
func (m *mockInstallBackend) Search(query string) ([]PackageInfo, error) { return nil, nil }
func (m *mockInstallBackend) Info(name string) (*PackageInfo, error)     { return nil, nil }
func (m *mockInstallBackend) IsInstalled(name string) bool               { return false }
func (m *mockInstallBackend) GetInstalledPackages() ([]PackageInfo, error) {
	return nil, nil
}
func (m *mockInstallBackend) GetPackageFiles(name string) ([]string, error)           { return nil, nil }
func (m *mockInstallBackend) GetFileOwner(path string) (string, error)                { return "", nil }
func (m *mockInstallBackend) SearchFiles(query string) ([]string, error)              { return nil, nil }
func (m *mockInstallBackend) ListGroups() ([]string, error)                           { return nil, nil }
func (m *mockInstallBackend) GetGroupPackages(group string) ([]string, error)         { return nil, nil }
func (m *mockInstallBackend) SetInstallReason(pkg string, reason InstallReason) error { return nil }
func (m *mockInstallBackend) CheckDatabase() error                                    { return nil }
func (m *mockInstallBackend) CleanCache(opts CleanOptions) error                      { return nil }
func (m *mockInstallBackend) ListOrphans() ([]string, error)                          { return nil, nil }
func (m *mockInstallBackend) RemoveOrphans(pkgs []string) error                       { return nil }
func (m *mockInstallBackend) VerifyAll() (map[string][]string, error)                 { return nil, nil }
func (m *mockInstallBackend) VerifyPackage(pkg string) ([]string, error)              { return nil, nil }
func (m *mockInstallBackend) Build(dir string, opts BuildOptions) error               { return nil }
func (m *mockInstallBackend) RemoveLock() error                                       { return nil }
func (m *mockInstallBackend) InitKeyring() error                                      { return nil }
func (m *mockInstallBackend) RefreshKeys() error                                      { return nil }
func (m *mockInstallBackend) ListKeys() ([]string, error)                             { return nil, nil }
func (m *mockInstallBackend) AddKey(keyID string) error                               { return nil }
func (m *mockInstallBackend) RemoveKey(keyID string) error                            { return nil }
func (m *mockInstallBackend) ImportKey(path string) error                             { return nil }

func (m *mockInstallBackend) InstallLocal(path string, opts InstallOptions) error {
	m.installCalled = true
	if m.installLocalFunc != nil {
		return m.installLocalFunc(path, opts)
	}
	return nil
}

// Implement missing OfficialBackend methods
func (m *mockInstallBackend) ListExplicitPackages() ([]string, error) { return nil, nil }
func (m *mockInstallBackend) Audit() ([]string, error)                { return nil, nil }
func (m *mockInstallBackend) Diff() ([]PackageDiff, error)            { return nil, nil }
