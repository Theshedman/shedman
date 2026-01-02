package installer_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theshedman/shedman/pkg/shedman/installer"
)

func TestAURInstaller_NewAURInstaller(t *testing.T) {
	ai := installer.NewAURInstaller()
	if ai == nil {
		t.Fatal("NewAURInstaller should return non-nil")
	}
}

func TestAURInstaller_Clone_FirstTime(t *testing.T) {
	ai := installer.NewAURInstaller()

	var executedCmds [][]string
	ai.SetExecutor(func(cmd []string) error {
		executedCmds = append(executedCmds, cmd)
		return nil
	})

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
	ai := installer.NewAURInstaller()

	// Create fake existing clone with .git directory
	tmpDir := t.TempDir()
	ai.SetCacheDir(tmpDir)
	pkgDir := filepath.Join(tmpDir, "neovim-nightly")
	gitDir := filepath.Join(pkgDir, ".git")
	os.MkdirAll(gitDir, 0755)
	os.WriteFile(filepath.Join(gitDir, "config"), []byte(""), 0644)

	var executedCmds [][]string
	ai.SetExecutor(func(cmd []string) error {
		executedCmds = append(executedCmds, cmd)
		return nil
	})

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
	ai := installer.NewAURInstaller()
	tmpDir := t.TempDir()
	ai.SetCacheDir(tmpDir)

	// Non-existent package
	if !ai.IsFirstTime("nonexistent-pkg") {
		t.Error("Should be first time for non-existent package")
	}

	// Create fake clone
	pkgDir := filepath.Join(tmpDir, "existing-pkg")
	os.MkdirAll(filepath.Join(pkgDir, ".git"), 0755)

	if ai.IsFirstTime("existing-pkg") {
		t.Error("Should not be first time for existing package")
	}
}

func TestAURInstaller_GetPKGBUILD(t *testing.T) {
	ai := installer.NewAURInstaller()
	tmpDir := t.TempDir()
	ai.SetCacheDir(tmpDir)

	// Create fake PKGBUILD
	pkgDir := filepath.Join(tmpDir, "test-pkg")
	os.MkdirAll(pkgDir, 0755)
	expectedContent := "pkgname=test-pkg\npkgver=1.0.0"
	os.WriteFile(filepath.Join(pkgDir, "PKGBUILD"), []byte(expectedContent), 0644)

	content, err := ai.GetPKGBUILD("test-pkg")
	if err != nil {
		t.Fatalf("GetPKGBUILD failed: %v", err)
	}

	if content != expectedContent {
		t.Errorf("Expected %q, got %q", expectedContent, content)
	}
}

func TestAURInstaller_GetPKGBUILDDiff(t *testing.T) {
	ai := installer.NewAURInstaller()
	tmpDir := t.TempDir()
	ai.SetCacheDir(tmpDir)

	// Create fake package dir with git
	pkgDir := filepath.Join(tmpDir, "test-pkg")
	os.MkdirAll(filepath.Join(pkgDir, ".git"), 0755)
	os.WriteFile(filepath.Join(pkgDir, "PKGBUILD"), []byte("pkgver=2.0"), 0644)

	var executedCmd []string
	ai.SetExecutor(func(cmd []string) error {
		executedCmd = cmd
		return nil
	})

	_, err := ai.GetPKGBUILDDiff("test-pkg")
	if err != nil {
		t.Fatalf("GetPKGBUILDDiff failed: %v", err)
	}

	// Should run git diff
	cmdStr := strings.Join(executedCmd, " ")
	if !strings.Contains(cmdStr, "diff") {
		t.Errorf("Expected git diff command, got %v", executedCmd)
	}
}

func TestAURInstaller_VerifyChecksums(t *testing.T) {
	ai := installer.NewAURInstaller()

	var executedCmd []string
	ai.SetExecutor(func(cmd []string) error {
		executedCmd = cmd
		return nil
	})

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
	ai := installer.NewAURInstaller()
	ai.SetSandboxEnabled(true)

	var executedCmd []string
	ai.SetExecutor(func(cmd []string) error {
		executedCmd = cmd
		return nil
	})

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
	ai := installer.NewAURInstaller()
	ai.SetSandboxEnabled(false)

	var executedCmd []string
	ai.SetExecutor(func(cmd []string) error {
		executedCmd = cmd
		return nil
	})

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
	ai := installer.NewAURInstaller()
	tmpDir := t.TempDir()
	ai.SetCacheDir(tmpDir)

	// Create fake built package
	pkgDir := filepath.Join(tmpDir, "test-pkg")
	os.MkdirAll(pkgDir, 0755)
	os.WriteFile(filepath.Join(pkgDir, "test-pkg-1.0-1-x86_64.pkg.tar.zst"), []byte("fake"), 0644)

	var executedCmd []string
	ai.SetExecutor(func(cmd []string) error {
		executedCmd = cmd
		return nil
	})

	err := ai.Install("test-pkg")
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Should run pacman -U
	cmdStr := strings.Join(executedCmd, " ")
	if !strings.Contains(cmdStr, "pacman") || !strings.Contains(cmdStr, "-U") {
		t.Errorf("Expected 'pacman -U' command, got %v", executedCmd)
	}
}
