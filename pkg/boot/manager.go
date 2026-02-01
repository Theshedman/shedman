package boot

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/theshedman/shedman/pkg/core"
)

// Executor defines command execution for boot management
type Executor interface {
	CombinedOutput(name string, args ...string) ([]byte, error)
	LookPath(file string) (string, error)
}

// RealExecutor implements Executor using os/exec
type RealExecutor struct{}

func (e *RealExecutor) CombinedOutput(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

func (e *RealExecutor) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

// Manager handles boot management operations
type Manager struct {
	core            *core.Engine
	exec            Executor
	grubConfigPaths []string
}

// NewWithExecutor creates a new boot manager with a custom executor
func NewWithExecutor(c *core.Engine, exec Executor) *Manager {
	return &Manager{
		core:            c,
		exec:            exec,
		grubConfigPaths: []string{"/boot/grub/grub.cfg", "/grub/grub.cfg"},
	}
}

// New creates a new boot manager
func New(c *core.Engine) *Manager {
	return &Manager{
		core:            c,
		exec:            &RealExecutor{},
		grubConfigPaths: []string{"/boot/grub/grub.cfg", "/grub/grub.cfg"},
	}
}

// Kernel represents a kernel
type Kernel struct {
	Name    string
	Version string
	Current bool
}

var knownKernels = []string{
	"linux",
	"linux-lts",
	"linux-zen",
	"linux-hardened",
}

// List lists available kernels (installed ones)
func (m *Manager) List() ([]Kernel, error) {
	var kernels []Kernel
	current := m.currentKernel()

	for _, name := range knownKernels {
		if m.core.IsInstalled(name) {
			info, err := m.core.Info(name)
			version := "unknown"
			if err == nil && info != nil {
				version = info.Version
			}

			kernels = append(kernels, Kernel{
				Name:    name,
				Version: version,
				Current: name == current,
			})
		}
	}

	return kernels, nil
}

// SetDefault sets the default kernel
func (m *Manager) SetDefault(kernel string) error {
	// Validate kernel is installed
	if !m.core.IsInstalled(kernel) {
		return fmt.Errorf("kernel %s is not installed", kernel)
	}

	// Check for systemd-boot
	if _, err := m.exec.LookPath("bootctl"); err == nil {
		out, err := m.exec.CombinedOutput("bootctl", "set-default", kernel+".conf")
		if err != nil {
			return fmt.Errorf("bootctl failed (entry '%s.conf' might not exist): %w\nOutput: %s", kernel, err, string(out))
		}
		return nil
	}

	// GRUB check
	if _, err := m.exec.LookPath("grub-set-default"); err == nil {
		entry, err := m.findGrubEntry(kernel)
		if err != nil {
			return err
		}
		out, err := m.exec.CombinedOutput("grub-set-default", entry)
		if err != nil {
			return fmt.Errorf("grub-set-default failed: %w\nOutput: %s", err, string(out))
		}
		return nil
	}

	return fmt.Errorf("no supported bootloader management tool found (bootctl/grub-set-default)")
}

// SetOneshot sets the next boot to use a specific kernel once.
func (m *Manager) SetOneshot(kernel string) error {
	// Validate kernel is installed
	if !m.core.IsInstalled(kernel) {
		return fmt.Errorf("kernel %s is not installed", kernel)
	}

	// Check for systemd-boot
	if _, err := m.exec.LookPath("bootctl"); err == nil {
		out, err := m.exec.CombinedOutput("bootctl", "set-oneshot", kernel+".conf")
		if err != nil {
			return fmt.Errorf("bootctl failed (entry '%s.conf' might not exist): %w\nOutput: %s", kernel, err, string(out))
		}
		return nil
	}

	// GRUB oneshot (grub-reboot)
	if _, err := m.exec.LookPath("grub-reboot"); err == nil {
		entry, err := m.findGrubEntry(kernel)
		if err != nil {
			return err
		}
		out, err := m.exec.CombinedOutput("grub-reboot", entry)
		if err != nil {
			return fmt.Errorf("grub-reboot failed: %w\nOutput: %s", err, string(out))
		}
		return nil
	}

	return fmt.Errorf("no supported bootloader management tool found (bootctl/grub-reboot)")
}

func (m *Manager) currentKernel() string {
	if m.exec == nil {
		return ""
	}
	out, err := m.exec.CombinedOutput("uname", "-r")
	if err != nil {
		return ""
	}
	return kernelFromUname(strings.TrimSpace(string(out)))
}

func kernelFromUname(uname string) string {
	lower := strings.ToLower(uname)
	switch {
	case strings.Contains(lower, "lts"):
		return "linux-lts"
	case strings.Contains(lower, "zen"):
		return "linux-zen"
	case strings.Contains(lower, "hardened"):
		return "linux-hardened"
	case lower != "":
		return "linux"
	default:
		return ""
	}
}

func (m *Manager) findGrubEntry(kernel string) (string, error) {
	for _, path := range m.grubConfigPaths {
		entry, err := parseGrubConfig(path, kernel)
		if err == nil && entry != "" {
			return entry, nil
		}
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
	}
	return "", fmt.Errorf("failed to locate grub entry for kernel %s", kernel)
}

type grubEntry struct {
	fullTitle  string
	depth      int
	isFallback bool
	matches    bool
}

type submenuEntry struct {
	title string
	depth int
}

func parseGrubConfig(path, kernel string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	menuRe := regexpMenuEntry
	subRe := regexpSubmenuEntry
	kernelToken := "vmlinuz-" + kernel

	var (
		braceDepth int
		stack      []submenuEntry
		current    *grubEntry
		primary    string
		fallback   string
	)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		open := strings.Count(line, "{")
		close := strings.Count(line, "}")

		if current != nil && strings.Contains(line, kernelToken) {
			current.matches = true
		}

		if matches := subRe.FindStringSubmatch(line); len(matches) == 2 {
			subDepth := braceDepth + open - close
			stack = append(stack, submenuEntry{
				title: matches[1],
				depth: subDepth,
			})
		}

		if matches := menuRe.FindStringSubmatch(line); len(matches) == 2 {
			fullTitle := matches[1]
			if len(stack) > 0 {
				var parts []string
				for _, s := range stack {
					parts = append(parts, s.title)
				}
				fullTitle = strings.Join(parts, ">") + ">" + fullTitle
			}
			entryDepth := braceDepth + open - close
			current = &grubEntry{
				fullTitle:  fullTitle,
				depth:      entryDepth,
				isFallback: strings.Contains(strings.ToLower(fullTitle), "fallback"),
			}
		}

		braceDepth += open - close

		if current != nil && braceDepth < current.depth {
			if current.matches {
				if !current.isFallback && primary == "" {
					primary = current.fullTitle
				} else if fallback == "" {
					fallback = current.fullTitle
				}
			}
			current = nil
		}

		for len(stack) > 0 && braceDepth < stack[len(stack)-1].depth {
			stack = stack[:len(stack)-1]
		}
	}

	if current != nil && current.matches {
		if !current.isFallback && primary == "" {
			primary = current.fullTitle
		} else if fallback == "" {
			fallback = current.fullTitle
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	if primary != "" {
		return primary, nil
	}
	if fallback != "" {
		return fallback, nil
	}
	return "", fmt.Errorf("no matching grub entry found in %s", path)
}

var (
	regexpMenuEntry    = regexp.MustCompile(`^menuentry ['"]([^'"]+)['"]`)
	regexpSubmenuEntry = regexp.MustCompile(`^submenu ['"]([^'"]+)['"]`)
)
