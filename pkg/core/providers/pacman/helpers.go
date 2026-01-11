package pacman

import (
	"github.com/theshedman/shedman/internal/alpm"
	"github.com/theshedman/shedman/pkg/core"
)

// PackageToInfo converts an alpm.AlpmPackage to core.PackageInfo
func PackageToInfo(pkg alpm.AlpmPackage) core.PackageInfo {
	return core.PackageInfo{
		Name:          pkg.Name(),
		Version:       pkg.Version(),
		Description:   pkg.Description(),
		Source:        core.SourceOfficial, // Default to official
		PackageType:   core.PackageTypeArch,
		Depends:       pkg.Depends().Slice(),
		OptDepends:    pkg.OptionalDepends().Slice(),
		Provides:      pkg.Provides().Slice(),
		Conflicts:     pkg.Conflicts().Slice(),
		Size:          pkg.Size(),
		InstalledSize: pkg.ISize(),
	}
}
