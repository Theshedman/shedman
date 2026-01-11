package resolver

import (
	"fmt"

	"github.com/theshedman/shedman/pkg/core/pkgdb"
)

// UpgradeInfo represents a package upgrade
type UpgradeInfo struct {
	Package    pkgdb.PackageInfo
	OldVersion string
}

// RemovalInfo represents a package to be removed
type RemovalInfo struct {
	Name   string
	Reason string
	Size   int64 // Size that will be freed
}

// InstallSummary collects and summarizes installation changes
type InstallSummary struct {
	installs   []pkgdb.PackageInfo
	reinstalls []pkgdb.PackageInfo // Packages being reinstalled
	upgrades   []UpgradeInfo
	removals   []RemovalInfo
}

// NewInstallSummary creates a new InstallSummary
func NewInstallSummary() *InstallSummary {
	return &InstallSummary{
		installs:   make([]pkgdb.PackageInfo, 0),
		reinstalls: make([]pkgdb.PackageInfo, 0),
		upgrades:   make([]UpgradeInfo, 0),
		removals:   make([]RemovalInfo, 0),
	}
}

// AddInstall adds a package to be installed
func (s *InstallSummary) AddInstall(pkg pkgdb.PackageInfo) {
	s.installs = append(s.installs, pkg)
}

// AddReinstall adds a package to be reinstalled
func (s *InstallSummary) AddReinstall(pkg pkgdb.PackageInfo) {
	s.reinstalls = append(s.reinstalls, pkg)
}

// AddUpgrade adds a package to be upgraded
func (s *InstallSummary) AddUpgrade(pkg pkgdb.PackageInfo, oldVersion string) {
	s.upgrades = append(s.upgrades, UpgradeInfo{
		Package:    pkg,
		OldVersion: oldVersion,
	})
}

// AddRemoval adds a package to be removed
func (s *InstallSummary) AddRemoval(name, reason string) {
	s.removals = append(s.removals, RemovalInfo{
		Name:   name,
		Reason: reason,
	})
}

// AddRemovalWithSize adds a package to be removed with its size
func (s *InstallSummary) AddRemovalWithSize(name, reason string, size int64) {
	s.removals = append(s.removals, RemovalInfo{
		Name:   name,
		Reason: reason,
		Size:   size,
	})
}

// InstallCount returns the number of packages to install
func (s *InstallSummary) InstallCount() int {
	return len(s.installs)
}

// UpgradeCount returns the number of packages to upgrade
func (s *InstallSummary) UpgradeCount() int {
	return len(s.upgrades)
}

// RemovalCount returns the number of packages to remove
func (s *InstallSummary) RemovalCount() int {
	return len(s.removals)
}

// GetInstalls returns packages to install
func (s *InstallSummary) GetInstalls() []pkgdb.PackageInfo {
	return s.installs
}

// GetUpgrades returns packages to upgrade
func (s *InstallSummary) GetUpgrades() []UpgradeInfo {
	return s.upgrades
}

// GetRemovals returns packages to remove
func (s *InstallSummary) GetRemovals() []RemovalInfo {
	return s.removals
}

// TotalDownloadSize returns the total download size in bytes
func (s *InstallSummary) TotalDownloadSize() int64 {
	var total int64
	for _, pkg := range s.installs {
		total += pkg.Size
	}
	for _, upgrade := range s.upgrades {
		total += upgrade.Package.Size
	}
	return total
}

// TotalInstalledSize returns the total installed size in bytes
func (s *InstallSummary) TotalInstalledSize() int64 {
	var total int64
	for _, pkg := range s.installs {
		total += pkg.InstalledSize
	}
	for _, upgrade := range s.upgrades {
		total += upgrade.Package.InstalledSize
	}
	return total
}

// TotalRemovalSize returns the total size to be freed in bytes
func (s *InstallSummary) TotalRemovalSize() int64 {
	var total int64
	for _, removal := range s.removals {
		total += removal.Size
	}
	return total
}

// NetDiskChange returns the net disk space change (positive = more space used)
func (s *InstallSummary) NetDiskChange() int64 {
	return s.TotalInstalledSize() - s.TotalRemovalSize()
}

// IsEmpty returns true if there are no changes
func (s *InstallSummary) IsEmpty() bool {
	return len(s.installs) == 0 && len(s.upgrades) == 0 && len(s.removals) == 0
}

// TotalPackages returns the total number of packages affected
func (s *InstallSummary) TotalPackages() int {
	return len(s.installs) + len(s.upgrades) + len(s.removals)
}

// GroupBySource groups install packages by their source
func (s *InstallSummary) GroupBySource() map[string][]pkgdb.PackageInfo {
	result := make(map[string][]pkgdb.PackageInfo)
	for _, pkg := range s.installs {
		result[pkg.Source] = append(result[pkg.Source], pkg)
	}
	return result
}

// HasConflictRemovals returns true if any removals are due to conflicts
func (s *InstallSummary) HasConflictRemovals() bool {
	for _, r := range s.removals {
		if r.Reason != "" {
			return true
		}
	}
	return false
}

// FormatSize formats bytes into human-readable string
func FormatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GiB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MiB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KiB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// FormatSizeChange formats a size change with +/- prefix
func FormatSizeChange(bytes int64) string {
	if bytes >= 0 {
		return "+" + FormatSize(bytes)
	}
	return "-" + FormatSize(-bytes)
}

// SummaryLine represents a line in the summary display
type SummaryLine struct {
	Label string
	Value string
}

// GetSummaryLines returns structured summary information for display
func (s *InstallSummary) GetSummaryLines() []SummaryLine {
	lines := make([]SummaryLine, 0)

	if len(s.installs) > 0 {
		lines = append(lines, SummaryLine{
			Label: "Packages to install",
			Value: fmt.Sprintf("%d", len(s.installs)),
		})
	}

	if len(s.upgrades) > 0 {
		lines = append(lines, SummaryLine{
			Label: "Packages to upgrade",
			Value: fmt.Sprintf("%d", len(s.upgrades)),
		})
	}

	if len(s.removals) > 0 {
		lines = append(lines, SummaryLine{
			Label: "Packages to remove",
			Value: fmt.Sprintf("%d", len(s.removals)),
		})
	}

	downloadSize := s.TotalDownloadSize()
	if downloadSize > 0 {
		lines = append(lines, SummaryLine{
			Label: "Total download size",
			Value: FormatSize(downloadSize),
		})
	}

	netChange := s.NetDiskChange()
	lines = append(lines, SummaryLine{
		Label: "Net disk space change",
		Value: FormatSizeChange(netChange),
	})

	return lines
}

// GetPackageList returns a formatted list of package names for display
func (s *InstallSummary) GetPackageList() []string {
	names := make([]string, 0, len(s.installs)+len(s.upgrades))
	for _, pkg := range s.installs {
		names = append(names, pkg.Name)
	}
	for _, upgrade := range s.upgrades {
		names = append(names, fmt.Sprintf("%s (%s -> %s)",
			upgrade.Package.Name, upgrade.OldVersion, upgrade.Package.Version))
	}
	return names
}
