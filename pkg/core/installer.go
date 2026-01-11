package core

import (
	"errors"
	"os"
	"os/exec"

	"strings"

	"github.com/theshedman/shedman/internal/config"
)

// ErrPacmanNotFound is returned when pacman is not available
var ErrPacmanNotFound = errors.New("pacman is required but not found in PATH")

// Executor is a function that executes a command
type Executor func(dir string, cmd []string) error

// DefaultExecutor runs commands using os/exec
func DefaultExecutor(dir string, cmd []string) error {
	if len(cmd) == 0 {
		return nil
	}
	c := exec.Command(cmd[0], cmd[1:]...)
	c.Dir = dir
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	return c.Run()
}

// ToBackendOptions converts Options to InstallOptions
func ToBackendOptions(opts Options) InstallOptions {
	return InstallOptions{
		Needed:       opts.Needed,
		AsDeps:       opts.AsDeps,
		AsExplicit:   opts.AsExplicit,
		NoConfirm:    opts.NoConfirm,
		DownloadOnly: opts.DownloadOnly,
		Overwrite:    opts.Overwrite,
	}
}

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
	Install(pkg PackageInfo, opts Options) error
	InstallMultiple(pkgs []PackageInfo, opts Options) error
}

// ProgressInstaller extends Installer with progress callback support
type ProgressInstaller interface {
	Installer
	InstallWithProgress(pkgs []PackageInfo, opts Options, callback ProgressCallback) error
}

// PackageRequest represents a parsed package argument
type PackageRequest struct {
	Name    string
	Version string
	IsGroup bool
}

// ParsePackageRequest parses a package argument like "name@version" or "@group"
func ParsePackageRequest(arg string) PackageRequest {
	// Check for group
	if strings.HasPrefix(arg, "@") {
		return PackageRequest{Name: arg[1:], IsGroup: true}
	}
	// Check for version
	if idx := strings.LastIndex(arg, "@"); idx != -1 {
		return PackageRequest{
			Name:    arg[:idx],
			Version: arg[idx+1:],
		}
	}
	return PackageRequest{Name: arg}
}
