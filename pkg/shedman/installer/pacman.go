package installer

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/theshedman/shedman/pkg/shedman/pkgdb"
)

// ErrPacmanNotFound is returned when pacman is not available
var ErrPacmanNotFound = errors.New("pacman is required but not found in PATH")

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

// IsPacmanAvailable checks if pacman command exists in PATH
func (p *PacmanInstaller) IsPacmanAvailable() bool {
	_, err := exec.LookPath("pacman")
	return err == nil
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

// InstallWithProgress installs packages with real-time progress callbacks
func (p *PacmanInstaller) InstallWithProgress(pkgs []pkgdb.PackageInfo, opts Options, callback ProgressCallback) error {
	names := make([]string, len(pkgs))
	for i, pkg := range pkgs {
		names[i] = pkg.Name
	}
	cmdArgs := p.BuildCommand(names, opts)

	if len(cmdArgs) == 0 {
		return nil
	}

	// Create progress parser
	progress := NewPacmanProgress(callback)
	progress.SetTotalPackages(len(pkgs))

	// Execute with progress parsing
	return ProgressExecutor(cmdArgs, progress)
}

// ProgressExecutor runs a command and pipes output through the progress parser
func ProgressExecutor(cmd []string, progress *PacmanProgress) error {
	if len(cmd) == 0 {
		return nil
	}

	c := exec.Command(cmd[0], cmd[1:]...)
	c.Stdin = os.Stdin

	// Create pipes for stdout and stderr
	stdout, err := c.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := c.StderrPipe()
	if err != nil {
		return err
	}

	if err := c.Start(); err != nil {
		return err
	}

	// Parse stdout in goroutine
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Println(line) // Still print to terminal
			if progress != nil {
				progress.ParseLine(line)
			}
		}
	}()

	// Parse stderr in goroutine
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Fprintln(os.Stderr, line) // Still print to terminal
			if progress != nil {
				progress.ParseLine(line)
			}
		}
	}()

	return c.Wait()
}
