package installer

import (
	"github.com/theshedman/shedman/pkg/shedman/config"
	"github.com/theshedman/shedman/pkg/shedman/pkgdb"
)

// Executor is a function that executes a command
type Executor func(cmd []string) error

// Options holds installation options
type Options struct {
	Needed            bool
	AsDeps            bool
	AsExplicit        bool
	NoConfirm         bool
	DownloadOnly      bool
	Overwrite         string
	ParallelDownloads int  // Number of parallel downloads (from config)
	Verbose           bool // Verbose output
}

// DefaultOptions returns sensible defaults
func DefaultOptions() Options {
	return Options{
		AsExplicit:        true,
		ParallelDownloads: 5,
	}
}

// OptionsFromConfig creates Options from config settings
func OptionsFromConfig(cfg *config.Config) Options {
	opts := DefaultOptions()
	if cfg != nil {
		opts.ParallelDownloads = cfg.Network.ParallelDownloads
		opts.NoConfirm = !cfg.General.Confirm
	}
	return opts
}

// WithNeeded returns options with Needed flag set
func (o Options) WithNeeded() Options {
	o.Needed = true
	return o
}

// WithAsDeps returns options with AsDeps flag set
func (o Options) WithAsDeps() Options {
	o.AsDeps = true
	o.AsExplicit = false
	return o
}

// WithDownloadOnly returns options with DownloadOnly flag set
func (o Options) WithDownloadOnly() Options {
	o.DownloadOnly = true
	return o
}

// WithNoConfirm returns options with NoConfirm flag set
func (o Options) WithNoConfirm() Options {
	o.NoConfirm = true
	return o
}

// WithOverwrite returns options with Overwrite pattern set
func (o Options) WithOverwrite(pattern string) Options {
	o.Overwrite = pattern
	return o
}

// Installer is the interface for package installers
type Installer interface {
	Install(pkg pkgdb.PackageInfo, opts Options) error
	InstallMultiple(pkgs []pkgdb.PackageInfo, opts Options) error
}

// ProgressInstaller extends Installer with progress callback support
type ProgressInstaller interface {
	Installer
	InstallWithProgress(pkgs []pkgdb.PackageInfo, opts Options, callback ProgressCallback) error
}
