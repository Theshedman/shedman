package installer

import (
	"github.com/theshedman/shedman/pkg/shedman/pkgdb"
)

// Executor is a function that executes a command
type Executor func(cmd []string) error

// Options holds installation options
type Options struct {
	Needed       bool
	AsDeps       bool
	AsExplicit   bool
	NoConfirm    bool
	DownloadOnly bool
	Overwrite    string
}

// DefaultOptions returns sensible defaults
func DefaultOptions() Options {
	return Options{
		AsExplicit: true,
	}
}

// Installer is the interface for package installers
type Installer interface {
	Install(pkg pkgdb.PackageInfo, opts Options) error
	InstallMultiple(pkgs []pkgdb.PackageInfo, opts Options) error
}
