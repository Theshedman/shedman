package cmd

import (
	"bytes"
	"testing"

	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/pkg/core"
)

func TestInstallCommand_Exists(t *testing.T) {
	installCmd := InstallCmd
	if installCmd == nil {
		t.Fatal("Install command should exist")
	}

	if installCmd.Use != "install [packages...]" {
		t.Errorf("Expected Use 'install [packages...]', got '%s'", installCmd.Use)
	}
}

func TestInstallCommand_HasRequiredFlags(t *testing.T) {
	installCmd := InstallCmd

	flags := []string{"needed", "asdeps", "asexplicit", "aur", "official", "shedos", "overwrite"}

	for _, flag := range flags {
		if installCmd.Flags().Lookup(flag) == nil {
			t.Errorf("Missing flag: --%s", flag)
		}
	}
}

func TestInstallCommand_RequiresArgs(t *testing.T) {
	installCmd := InstallCmd

	// Install command requires at least 1 package
	if installCmd.Args == nil {
		t.Error("Install command should have Args validation")
	}
}

func TestInstallCommand_ShortDescription(t *testing.T) {
	installCmd := InstallCmd

	if installCmd.Short != "Install packages" {
		t.Errorf("Expected Short 'Install packages', got '%s'", installCmd.Short)
	}
}

func TestRunInstall_Group(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	// Mock Info for packages in @base
	// @base contains: base, linux, linux-firmware (from modified groups.go)

	pkgInfo := map[string]*core.PackageInfo{
		"base":           {Name: "base", Version: "1.0", Source: core.SourceOfficial},
		"base-devel":     {Name: "base-devel", Version: "1.0", Source: core.SourceOfficial},
		"linux":          {Name: "linux", Version: "6.0", Source: core.SourceOfficial},
		"linux-firmware": {Name: "linux-firmware", Version: "2023", Source: core.SourceOfficial},
		"networkmanager": {Name: "networkmanager", Version: "1.44", Source: core.SourceOfficial},
		"sudo":           {Name: "sudo", Version: "1.9", Source: core.SourceOfficial},
		"vim":            {Name: "vim", Version: "9.0", Source: core.SourceOfficial},
		"git":            {Name: "git", Version: "2.40", Source: core.SourceOfficial},
	}

	mock.InfoFunc = func(name string) (*core.PackageInfo, error) {
		if p, ok := pkgInfo[name]; ok {
			return p, nil
		}
		return nil, core.ErrPackageNotFound
	}

	installCalled := false
	installedPkgs := []string{}
	mock.InstallFunc = func(pkgs []string, opts core.InstallOptions) error {
		installCalled = true
		installedPkgs = pkgs
		return nil
	}

	var buf bytes.Buffer
	// Create InstallFlags
	flags := InstallFlags{
		Quiet:  true,
		Yes:    true,
		DryRun: false,
	}

	err := RunInstall(eng, []string{"@base"}, flags, &buf, config.Default())
	if err != nil {
		t.Fatalf("RunInstall failed: %v", err)
	}

	if !installCalled {
		t.Fatal("Install was not called")
	}

	// Verify all base packages were installed
	expected := []string{"base", "base-devel", "linux", "linux-firmware", "networkmanager", "sudo", "vim", "git"}
	for _, exp := range expected {
		found := false
		for _, inst := range installedPkgs {
			if inst == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected package %s was not installed", exp)
		}
	}
}
