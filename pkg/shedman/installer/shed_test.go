package installer_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theshedman/shedman/pkg/shedman/installer"
)

func TestShedInstaller_NewShedInstaller(t *testing.T) {
	si := installer.NewShedInstaller()
	if si == nil {
		t.Fatal("NewShedInstaller should return non-nil")
	}
}

func TestShedInstaller_Extract(t *testing.T) {
	si := installer.NewShedInstaller()
	tmpDir := t.TempDir()

	// Create a fake .shed file
	shedFile := filepath.Join(tmpDir, "test-pkg-1.0.shed")
	os.WriteFile(shedFile, []byte("fake shed content"), 0644)

	var executedCmd []string
	si.SetExecutor(func(cmd []string) error {
		executedCmd = cmd
		return nil
	})

	destDir := filepath.Join(tmpDir, "extracted")
	err := si.Extract(shedFile, destDir)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Should use tar or similar to extract
	cmdStr := strings.Join(executedCmd, " ")
	if !strings.Contains(cmdStr, "tar") && !strings.Contains(cmdStr, "unzip") {
		t.Errorf("Expected tar/unzip command, got %v", executedCmd)
	}
}

func TestShedInstaller_ReadManifest(t *testing.T) {
	si := installer.NewShedInstaller()
	tmpDir := t.TempDir()

	// Create fake extracted package with manifest
	pkgDir := filepath.Join(tmpDir, "test-pkg")
	os.MkdirAll(pkgDir, 0755)

	manifestContent := `name = "test-pkg"
version = "1.0.0"
description = "A test package"
`
	os.WriteFile(filepath.Join(pkgDir, "manifest.toml"), []byte(manifestContent), 0644)

	manifest, err := si.ReadManifest(pkgDir)
	if err != nil {
		t.Fatalf("ReadManifest failed: %v", err)
	}

	if manifest.Name != "test-pkg" {
		t.Errorf("Expected name 'test-pkg', got '%s'", manifest.Name)
	}
	if manifest.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", manifest.Version)
	}
}

func TestShedInstaller_VerifySignature(t *testing.T) {
	si := installer.NewShedInstaller()
	tmpDir := t.TempDir()

	// Create fake package and signature files
	shedFile := filepath.Join(tmpDir, "package.shed")
	sigFile := filepath.Join(tmpDir, "package.shed.sig")
	os.WriteFile(shedFile, []byte("fake package"), 0644)
	os.WriteFile(sigFile, []byte("fake signature"), 0644)

	var executedCmd []string
	si.SetExecutor(func(cmd []string) error {
		executedCmd = cmd
		return nil
	})

	err := si.VerifySignature(shedFile, sigFile)
	if err != nil {
		t.Fatalf("VerifySignature failed: %v", err)
	}

	// Should use gpg to verify
	cmdStr := strings.Join(executedCmd, " ")
	if !strings.Contains(cmdStr, "gpg") {
		t.Errorf("Expected gpg command, got %v", executedCmd)
	}
}

func TestShedInstaller_InstallFiles(t *testing.T) {
	si := installer.NewShedInstaller()
	tmpDir := t.TempDir()

	// Create fake extracted package
	pkgDir := filepath.Join(tmpDir, "test-pkg")
	filesDir := filepath.Join(pkgDir, "files")
	os.MkdirAll(filepath.Join(filesDir, "usr", "bin"), 0755)
	os.WriteFile(filepath.Join(filesDir, "usr", "bin", "test-app"), []byte("#!/bin/bash\necho hello"), 0755)

	var executedCmds [][]string
	si.SetExecutor(func(cmd []string) error {
		executedCmds = append(executedCmds, cmd)
		return nil
	})

	err := si.InstallFiles(pkgDir, "/")
	if err != nil {
		t.Fatalf("InstallFiles failed: %v", err)
	}

	// Should copy files to destination
	found := false
	for _, cmd := range executedCmds {
		cmdStr := strings.Join(cmd, " ")
		if strings.Contains(cmdStr, "cp") || strings.Contains(cmdStr, "rsync") || strings.Contains(cmdStr, "install") {
			found = true
		}
	}
	if !found {
		t.Errorf("Expected cp/rsync/install command, got %v", executedCmds)
	}
}

func TestShedInstaller_RunHooks(t *testing.T) {
	si := installer.NewShedInstaller()
	tmpDir := t.TempDir()

	// Create fake package with hooks
	pkgDir := filepath.Join(tmpDir, "test-pkg")
	hooksDir := filepath.Join(pkgDir, "hooks")
	os.MkdirAll(hooksDir, 0755)
	os.WriteFile(filepath.Join(hooksDir, "post-install.sh"), []byte("#!/bin/bash\necho installed"), 0755)

	var executedCmd []string
	si.SetExecutor(func(cmd []string) error {
		executedCmd = cmd
		return nil
	})

	err := si.RunHooks(pkgDir, "post-install")
	if err != nil {
		t.Fatalf("RunHooks failed: %v", err)
	}

	// Should execute the hook script
	if len(executedCmd) == 0 {
		t.Error("Expected hook to be executed")
	}
}

func TestShedInstaller_Install_FullFlow(t *testing.T) {
	si := installer.NewShedInstaller()
	tmpDir := t.TempDir()
	si.SetCacheDir(tmpDir)

	// Create a fake .shed file
	shedFile := filepath.Join(tmpDir, "test-pkg-1.0.shed")
	os.WriteFile(shedFile, []byte("fake"), 0644)

	var executedCmds [][]string
	si.SetExecutor(func(cmd []string) error {
		executedCmds = append(executedCmds, cmd)
		return nil
	})

	// Install should orchestrate: extract, verify, install files, run hooks
	err := si.Install(shedFile)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	if len(executedCmds) == 0 {
		t.Error("Expected commands to be executed")
	}
}

func TestShedInstaller_Remove(t *testing.T) {
	si := installer.NewShedInstaller()
	tmpDir := t.TempDir()
	si.SetCacheDir(tmpDir)

	// Create installed package directory with manifest
	installedDir := filepath.Join(tmpDir, "installed", "test-pkg")
	hooksDir := filepath.Join(installedDir, "hooks")
	os.MkdirAll(hooksDir, 0755)

	// Create manifest with files list
	manifestContent := `name = "test-pkg"
version = "1.0.0"
description = "A test package"
files = ["/usr/bin/test-app", "/usr/share/test-pkg/data.txt"]
`
	os.WriteFile(filepath.Join(installedDir, "manifest.toml"), []byte(manifestContent), 0644)

	// Create pre-remove hook
	os.WriteFile(filepath.Join(hooksDir, "pre-remove.sh"), []byte("#!/bin/bash\necho pre-remove"), 0755)
	os.WriteFile(filepath.Join(hooksDir, "post-remove.sh"), []byte("#!/bin/bash\necho post-remove"), 0755)

	var executedCmds [][]string
	si.SetExecutor(func(cmd []string) error {
		executedCmds = append(executedCmds, cmd)
		return nil
	})

	// Remove should: run pre-remove hook, remove files, run post-remove hook, remove manifest
	err := si.Remove("test-pkg")
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	// Should have executed hooks
	if len(executedCmds) < 2 {
		t.Errorf("Expected at least 2 commands (pre/post-remove hooks), got %d", len(executedCmds))
	}
}

func TestShedInstaller_Remove_NotInstalled(t *testing.T) {
	si := installer.NewShedInstaller()
	tmpDir := t.TempDir()
	si.SetCacheDir(tmpDir)

	// Try to remove package that isn't installed
	err := si.Remove("nonexistent-pkg")
	if err == nil {
		t.Error("Expected error when removing non-installed package")
	}
}

func TestShedInstaller_IsInstalled(t *testing.T) {
	si := installer.NewShedInstaller()
	tmpDir := t.TempDir()
	si.SetCacheDir(tmpDir)

	// Create installed package
	installedDir := filepath.Join(tmpDir, "installed", "test-pkg")
	os.MkdirAll(installedDir, 0755)
	os.WriteFile(filepath.Join(installedDir, "manifest.toml"), []byte("name = \"test-pkg\"\nversion = \"1.0.0\""), 0644)

	if !si.IsInstalled("test-pkg") {
		t.Error("Expected IsInstalled to return true for installed package")
	}

	if si.IsInstalled("nonexistent-pkg") {
		t.Error("Expected IsInstalled to return false for non-installed package")
	}
}
