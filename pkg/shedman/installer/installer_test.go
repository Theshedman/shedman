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
