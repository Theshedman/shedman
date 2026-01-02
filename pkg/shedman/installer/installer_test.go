package installer_test

import (
	"testing"

	"github.com/theshedman/shedman/pkg/shedman/installer"
	"github.com/theshedman/shedman/pkg/shedman/pkgdb"
)

func TestInstallOptions_Defaults(t *testing.T) {
	opts := installer.DefaultOptions()

	if opts.AsExplicit != true {
		t.Error("Default should be AsExplicit=true")
	}
	if opts.Needed != false {
		t.Error("Default should be Needed=false")
	}
}

func TestInstaller_Interface(t *testing.T) {
	mock := &mockInstaller{}

	// Verify it satisfies the interface
	var i installer.Installer = mock

	pkg := pkgdb.PackageInfo{Name: "neovim", Version: "0.10.0"}
	err := i.Install(pkg, installer.DefaultOptions())

	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	if !mock.installCalled {
		t.Error("Install should have been called")
	}
}

func TestPacmanInstaller_BuildCommand(t *testing.T) {
	pi := installer.NewPacmanInstaller()

	opts := installer.Options{
		Needed:    true,
		AsDeps:    true,
		NoConfirm: true,
	}

	cmd := pi.BuildCommand([]string{"neovim", "git"}, opts)

	// Should contain pacman -S
	if cmd[0] != "pacman" || cmd[1] != "-S" {
		t.Error("Command should start with 'pacman -S'")
	}

	// Check flags are present
	hasNeeded := false
	hasAsDeps := false
	hasNoConfirm := false
	for _, arg := range cmd {
		if arg == "--needed" {
			hasNeeded = true
		}
		if arg == "--asdeps" {
			hasAsDeps = true
		}
		if arg == "--noconfirm" {
			hasNoConfirm = true
		}
	}

	if !hasNeeded {
		t.Error("Missing --needed flag")
	}
	if !hasAsDeps {
		t.Error("Missing --asdeps flag")
	}
	if !hasNoConfirm {
		t.Error("Missing --noconfirm flag")
	}
}

func TestPacmanInstaller_BuildCommand_WithPackages(t *testing.T) {
	pi := installer.NewPacmanInstaller()
	opts := installer.DefaultOptions()

	cmd := pi.BuildCommand([]string{"neovim", "git", "htop"}, opts)

	// Should end with package names
	if cmd[len(cmd)-3] != "neovim" || cmd[len(cmd)-2] != "git" || cmd[len(cmd)-1] != "htop" {
		t.Error("Command should end with package names")
	}
}

func TestPacmanInstaller_Execute_WithMockExecutor(t *testing.T) {
	pi := installer.NewPacmanInstaller()

	// Create a mock executor that records the command
	var executedCmd []string
	pi.SetExecutor(func(cmd []string) error {
		executedCmd = cmd
		return nil
	})

	pkgs := []pkgdb.PackageInfo{
		{Name: "neovim"},
	}
	opts := installer.Options{NoConfirm: true}

	err := pi.InstallMultiple(pkgs, opts)
	if err != nil {
		t.Fatalf("InstallMultiple failed: %v", err)
	}

	if len(executedCmd) == 0 {
		t.Error("Executor should have been called")
	}

	if executedCmd[0] != "pacman" {
		t.Error("Should call pacman")
	}
}

// Mock installer for testing
type mockInstaller struct {
	installCalled bool
}

func (m *mockInstaller) Install(pkg pkgdb.PackageInfo, opts installer.Options) error {
	m.installCalled = true
	return nil
}

func (m *mockInstaller) InstallMultiple(pkgs []pkgdb.PackageInfo, opts installer.Options) error {
	m.installCalled = true
	return nil
}
