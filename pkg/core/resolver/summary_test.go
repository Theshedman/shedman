package resolver_test

import (
	"testing"

	"github.com/theshedman/shedman/pkg/shedman/pkgdb"
	"github.com/theshedman/shedman/pkg/shedman/resolver"
)

func TestInstallSummary_BasicInfo(t *testing.T) {
	summary := resolver.NewInstallSummary()

	summary.AddInstall(pkgdb.PackageInfo{
		Name:          "neovim",
		Version:       "0.10.0",
		Source:        pkgdb.SourceOfficial,
		Size:          5000000,
		InstalledSize: 15000000,
	})

	if summary.InstallCount() != 1 {
		t.Errorf("Expected 1 install, got %d", summary.InstallCount())
	}
	if summary.TotalDownloadSize() != 5000000 {
		t.Errorf("Expected 5MB download, got %d", summary.TotalDownloadSize())
	}
	if summary.TotalInstalledSize() != 15000000 {
		t.Errorf("Expected 15MB installed, got %d", summary.TotalInstalledSize())
	}
}

func TestInstallSummary_Upgrades(t *testing.T) {
	summary := resolver.NewInstallSummary()

	summary.AddUpgrade(pkgdb.PackageInfo{
		Name:    "git",
		Version: "2.43.0",
	}, "2.42.0")

	if summary.UpgradeCount() != 1 {
		t.Errorf("Expected 1 upgrade, got %d", summary.UpgradeCount())
	}

	upgrades := summary.GetUpgrades()
	if len(upgrades) != 1 || upgrades[0].OldVersion != "2.42.0" {
		t.Error("Expected upgrade from 2.42.0")
	}
}

func TestInstallSummary_Removals(t *testing.T) {
	summary := resolver.NewInstallSummary()

	summary.AddRemoval("vim", "replaced by neovim")

	if summary.RemovalCount() != 1 {
		t.Errorf("Expected 1 removal, got %d", summary.RemovalCount())
	}
}

func TestInstallSummary_NetDiskChange(t *testing.T) {
	summary := resolver.NewInstallSummary()

	// Install 15MB
	summary.AddInstall(pkgdb.PackageInfo{
		Name:          "neovim",
		InstalledSize: 15000000,
	})

	// Remove 10MB
	summary.AddRemovalWithSize("vim", "replaced", 10000000)

	// Net change should be +5MB
	netChange := summary.NetDiskChange()
	if netChange != 5000000 {
		t.Errorf("Expected net +5MB, got %d", netChange)
	}
}

func TestInstallSummary_GroupBySource(t *testing.T) {
	summary := resolver.NewInstallSummary()

	summary.AddInstall(pkgdb.PackageInfo{Name: "git", Source: pkgdb.SourceOfficial})
	summary.AddInstall(pkgdb.PackageInfo{Name: "yay", Source: pkgdb.SourceAUR})
	summary.AddInstall(pkgdb.PackageInfo{Name: "curl", Source: pkgdb.SourceOfficial})

	grouped := summary.GroupBySource()
	if len(grouped[pkgdb.SourceOfficial]) != 2 {
		t.Errorf("Expected 2 official packages, got %d", len(grouped[pkgdb.SourceOfficial]))
	}
	if len(grouped[pkgdb.SourceAUR]) != 1 {
		t.Errorf("Expected 1 AUR package, got %d", len(grouped[pkgdb.SourceAUR]))
	}
}

func TestInstallSummary_IsEmpty(t *testing.T) {
	empty := resolver.NewInstallSummary()
	if !empty.IsEmpty() {
		t.Error("Expected empty summary")
	}

	nonEmpty := resolver.NewInstallSummary()
	nonEmpty.AddInstall(pkgdb.PackageInfo{Name: "git"})
	if nonEmpty.IsEmpty() {
		t.Error("Expected non-empty summary")
	}
}
