package core

import (
	"testing"

)

func TestInstallSummary_BasicInfo(t *testing.T) {
	summary := NewInstallSummary()

	summary.AddInstall(PackageInfo{
		Name:          "neovim",
		Version:       "0.10.0",
		Source:        SourceOfficial,
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
	summary := NewInstallSummary()

	summary.AddUpgrade(PackageInfo{
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
	summary := NewInstallSummary()

	summary.AddRemoval("vim", "replaced by neovim")

	if summary.RemovalCount() != 1 {
		t.Errorf("Expected 1 removal, got %d", summary.RemovalCount())
	}
}

func TestInstallSummary_NetDiskChange(t *testing.T) {
	summary := NewInstallSummary()

	// Install 15MB
	summary.AddInstall(PackageInfo{
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
	summary := NewInstallSummary()

	summary.AddInstall(PackageInfo{Name: "git", Source: SourceOfficial})
	summary.AddInstall(PackageInfo{Name: "yay", Source: SourceAUR})
	summary.AddInstall(PackageInfo{Name: "curl", Source: SourceOfficial})

	grouped := summary.GroupBySource()
	if len(grouped[SourceOfficial]) != 2 {
		t.Errorf("Expected 2 official packages, got %d", len(grouped[SourceOfficial]))
	}
	if len(grouped[SourceAUR]) != 1 {
		t.Errorf("Expected 1 AUR package, got %d", len(grouped[SourceAUR]))
	}
}

func TestInstallSummary_IsEmpty(t *testing.T) {
	empty := NewInstallSummary()
	if !empty.IsEmpty() {
		t.Error("Expected empty summary")
	}

	nonEmpty := NewInstallSummary()
	nonEmpty.AddInstall(PackageInfo{Name: "git"})
	if nonEmpty.IsEmpty() {
		t.Error("Expected non-empty summary")
	}
}
