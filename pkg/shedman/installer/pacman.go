package installer

import (
	"os"
	"os/exec"

	"github.com/theshedman/shedman/pkg/shedman/pkgdb"
)

// PacmanInstaller wraps pacman for official repo installs
type PacmanInstaller struct {
	executor Executor
}

// NewPacmanInstaller creates a new PacmanInstaller
func NewPacmanInstaller() *PacmanInstaller {
	return &PacmanInstaller{
		executor: DefaultExecutor,
	}
}

// DefaultExecutor runs commands using os/exec
func DefaultExecutor(cmd []string) error {
	if len(cmd) == 0 {
		return nil
	}
	c := exec.Command(cmd[0], cmd[1:]...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	return c.Run()
}

// SetExecutor sets a custom command executor (for testing)
func (p *PacmanInstaller) SetExecutor(exec Executor) {
	p.executor = exec
}

// BuildCommand builds the pacman command with flags
func (p *PacmanInstaller) BuildCommand(packages []string, opts Options) []string {
	cmd := []string{"pacman", "-S"}

	if opts.Needed {
		cmd = append(cmd, "--needed")
	}
	if opts.AsDeps {
		cmd = append(cmd, "--asdeps")
	}
	if opts.AsExplicit {
		cmd = append(cmd, "--asexplicit")
	}
	if opts.NoConfirm {
		cmd = append(cmd, "--noconfirm")
	}
	if opts.DownloadOnly {
		cmd = append(cmd, "--downloadonly")
	}
	if opts.Overwrite != "" {
		cmd = append(cmd, "--overwrite", opts.Overwrite)
	}

	cmd = append(cmd, packages...)
	return cmd
}

// Install installs a single package
func (p *PacmanInstaller) Install(pkg pkgdb.PackageInfo, opts Options) error {
	return p.InstallMultiple([]pkgdb.PackageInfo{pkg}, opts)
}

// InstallMultiple installs multiple packages
func (p *PacmanInstaller) InstallMultiple(pkgs []pkgdb.PackageInfo, opts Options) error {
	names := make([]string, len(pkgs))
	for i, pkg := range pkgs {
		names[i] = pkg.Name
	}
	cmd := p.BuildCommand(names, opts)
	if p.executor != nil {
		return p.executor(cmd)
	}
	return nil
}
