package pacman

import (
	"github.com/theshedman/shedman/internal/alpm"
	"github.com/theshedman/shedman/pkg/core"
)

// PackageToInfo converts an AlpmPackage to core.PackageInfo.
func PackageToInfo(pkg alpm.AlpmPackage) core.PackageInfo {
	info := core.PackageInfo{
		Name:          pkg.Name(),
		Version:       pkg.Version(),
		Description:   pkg.Description(),
		Source:        determineSource(pkg),
		Depends:       pkg.Depends().Slice(),
		OptDepends:    pkg.OptionalDepends().Slice(),
		Provides:      pkg.Provides().Slice(),
		Conflicts:     pkg.Conflicts().Slice(),
		Size:          pkg.Size(),
		InstalledSize: pkg.ISize(),
	}
	return info
}

// determineSource determines the package source based on the database name
func determineSource(pkg alpm.AlpmPackage) string {
	db := pkg.DB()
	if db == nil {
		return core.SourceOfficial
	}

	dbName := db.Name()
	switch dbName {
	case "local":
		// Local packages - check if they might be from AUR
		// (AUR packages are typically not in core/extra/multilib)
		return core.SourceOfficial // Default for installed packages
	case "core", "extra", "multilib", "community":
		return core.SourceOfficial
	case "aur":
		return core.SourceAUR
	case "shedos", "shedrepo":
		return core.SourceShedOS
	default:
		// Unknown repository, assume official
		return core.SourceOfficial
	}
}

// PackageToInfoWithDB converts an AlpmPackage to core.PackageInfo with explicit DB name
func PackageToInfoWithDB(pkg alpm.AlpmPackage, dbName string) core.PackageInfo {
	info := PackageToInfo(pkg)

	// Override source based on explicit DB name
	switch dbName {
	case "aur":
		info.Source = core.SourceAUR
	case "shedos", "shedrepo":
		info.Source = core.SourceShedOS
	case "core", "extra", "multilib", "community":
		info.Source = core.SourceOfficial
	}

	return info
}
