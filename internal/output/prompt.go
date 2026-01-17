package output

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// ConfirmOptions configures the confirmation prompt behavior
type ConfirmOptions struct {
	Default    bool          // Default value if user presses Enter
	SkipPrompt bool          // If true, return DefaultValue without prompting (for -y flag)
	Timeout    time.Duration // If > 0, auto-return Default after timeout
}

// stdinReader provides a safe way to read stdin with cancellation
type stdinReader struct {
	resultCh chan string
	errCh    chan error
	once     sync.Once
	started  bool
}

// newStdinReader creates a new stdin reader
func newStdinReader() *stdinReader {
	return &stdinReader{
		resultCh: make(chan string, 1),
		errCh:    make(chan error, 1),
	}
}

// start begins reading from stdin in a goroutine
func (r *stdinReader) start() {
	r.once.Do(func() {
		r.started = true
		go func() {
			reader := bufio.NewReader(os.Stdin)
			input, err := reader.ReadString('\n')
			if err != nil {
				select {
				case r.errCh <- err:
				default:
				}
				return
			}
			select {
			case r.resultCh <- input:
			default:
			}
		}()
	})
}

// IsTerminal checks if stdin is an interactive terminal
func IsTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// Confirm prompts the user for a yes/no confirmation
// Returns true if user confirms, false otherwise
func Confirm(message string, opts ConfirmOptions) bool {
	if opts.SkipPrompt {
		return opts.Default
	}

	// If not a terminal and timeout is set, use default immediately
	if !IsTerminal() && opts.Timeout > 0 {
		return opts.Default
	}

	// Build prompt suffix
	suffix := " [y/N]: "
	if opts.Default {
		suffix = " [Y/n]: "
	}

	// Add timeout indicator if applicable
	if opts.Timeout > 0 && IsTerminal() {
		suffix = fmt.Sprintf(" [%s in %ds]: ",
			map[bool]string{true: "Y/n", false: "y/N"}[opts.Default],
			int(opts.Timeout.Seconds()))
	}

	_, _ = fmt.Print(Colorize(Bold, message) + suffix)

	if opts.Timeout > 0 && IsTerminal() {
		return confirmWithTimeout(opts)
	}

	return readConfirmInput(opts.Default)
}

// readConfirmInput reads input from stdin and returns the confirmation result
func readConfirmInput(defaultVal bool) bool {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultVal
	}

	input = strings.TrimSpace(strings.ToLower(input))

	if input == "" {
		return defaultVal
	}

	return input == "y" || input == "yes"
}

// ReadInput prompts the user for string input
func ReadInput(prompt string) (string, error) {
	_, _ = fmt.Print(Colorize(Bold, prompt))

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

// confirmWithTimeout handles confirmation with a timeout
// Uses context for proper cancellation and cleanup
func confirmWithTimeout(opts ConfirmOptions) bool {
	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	reader := newStdinReader()
	reader.start()

	// Countdown ticker for visual feedback
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	remaining := int(opts.Timeout.Seconds())

	for {
		select {
		case input := <-reader.resultCh:
			input = strings.TrimSpace(strings.ToLower(input))
			if input == "" {
				return opts.Default
			}
			return input == "y" || input == "yes"

		case err := <-reader.errCh:
			if err != nil {
				return opts.Default
			}
			return opts.Default

		case <-ticker.C:
			remaining--
			if remaining > 0 && remaining <= 5 {
				// Show countdown for last 5 seconds
				_, _ = fmt.Printf(
					"\r%s [%s in %ds]: ",
					Colorize(Bold, ""),
					map[bool]string{true: "Y/n", false: "y/N"}[opts.Default],
					remaining)
			}

		case <-ctx.Done():
			// Clear line and show timeout message
			_, _ = fmt.Print(
				"\r")
			_, _ = fmt.Println(
				Colorize(Yellow, fmt.Sprintf("(timeout - proceeding with %s)",
					map[bool]string{true: "yes", false: "no"}[opts.Default])))
			return opts.Default
		}
	}
}

// ConfirmWithTimeout is a convenience wrapper for Confirm with timeout
func ConfirmWithTimeout(message string, defaultVal bool, timeout time.Duration) bool {
	return Confirm(message, ConfirmOptions{
		Default: defaultVal,
		Timeout: timeout,
	})
}

// ConfirmWithConfig creates confirm options from config values
func ConfirmWithConfig(skipPrompt bool, timeoutSeconds int) ConfirmOptions {
	return ConfirmOptions{
		Default:    true,
		SkipPrompt: skipPrompt,
		Timeout:    time.Duration(timeoutSeconds) * time.Second,
	}
}

// ConfirmInstall is a pre-configured confirm for installation
func ConfirmInstall(skipPrompt bool) bool {
	return Confirm("Proceed with installation?", ConfirmOptions{
		Default:    true,
		SkipPrompt: skipPrompt,
	})
}

// ConfirmInstallWithTimeout is ConfirmInstall with timeout support
func ConfirmInstallWithTimeout(skipPrompt bool, timeoutSeconds int) bool {
	if timeoutSeconds <= 0 {
		return ConfirmInstall(skipPrompt)
	}
	return Confirm("Proceed with installation?", ConfirmOptions{
		Default:    true,
		SkipPrompt: skipPrompt,
		Timeout:    time.Duration(timeoutSeconds) * time.Second,
	})
}

// ConfirmRemoval is a pre-configured confirm for removal
func ConfirmRemoval(pkgNames []string, skipPrompt bool) bool {
	msg := fmt.Sprintf("Remove %d package(s)?", len(pkgNames))
	return Confirm(msg, ConfirmOptions{
		Default:    false,
		SkipPrompt: skipPrompt,
	})
}

// ConfirmOverwrite is a pre-configured confirm for file overwrite
func ConfirmOverwrite(filePath string, skipPrompt bool) bool {
	msg := fmt.Sprintf("Overwrite %s?", filePath)
	return Confirm(msg, ConfirmOptions{
		Default:    false,
		SkipPrompt: skipPrompt,
	})
}

// ConfirmConflict is for confirming despite conflicts
func ConfirmConflict(conflictCount int, skipPrompt bool) bool {
	msg := fmt.Sprintf("%d conflict(s) detected. Continue anyway?", conflictCount)
	return Confirm(msg, ConfirmOptions{
		Default:    false,
		SkipPrompt: skipPrompt,
	})
}

// PrintSummary prints an installation summary with formatting
func PrintSummary(lines []SummaryLine) {
	if len(lines) == 0 {
		return
	}

	_, _ = fmt.Println()
	_, _ = fmt.Println(Colorize(Bold, "=== Installation Summary ==="))
	_, _ = fmt.Println()

	// Find max label width for alignment
	maxWidth := 0
	for _, line := range lines {
		if len(line.Label) > maxWidth {
			maxWidth = len(line.Label)
		}
	}

	for _, line := range lines {
		_, _ = fmt.Printf(
			"  %-*s : %s\n", maxWidth, line.Label, Colorize(Cyan, line.Value))
	}
	_, _ = fmt.Println()
}

// SummaryLine represents a line in the summary display
type SummaryLine = struct {
	Label string
	Value string
}

// PrintPackageList prints a list of packages with colors
func PrintPackageList(title string, packages []string, color string) {
	if len(packages) == 0 {
		return
	}

	_, _ = fmt.Println(
		Colorize(Bold, title+":"))
	for _, pkg := range packages {
		_, _ = fmt.Printf(
			"  %s %s\n", Colorize(color, "→"), pkg)
	}
	_, _ = fmt.Println()
}

// PrintInstallList prints packages to be installed
func PrintInstallList(packages []string) {
	PrintPackageList("Packages to install", packages, Green)
}

// PrintUpgradeList prints packages to be upgraded
func PrintUpgradeList(packages []string) {
	PrintPackageList("Packages to upgrade", packages, Blue)
}

// PrintRemovalList prints packages to be removed
func PrintRemovalList(packages []string) {
	PrintPackageList("Packages to remove", packages, Red)
}

// OptionalDepChoice represents an optional dependency option
type OptionalDepChoice struct {
	Name        string
	Description string
	Selected    bool
}

// SelectOptionalDeps prompts user to select optional dependencies
// Returns the names of selected dependencies
func SelectOptionalDeps(deps []OptionalDepChoice, skipPrompt bool) []string {
	if len(deps) == 0 {
		return nil
	}

	if skipPrompt {
		// Return none by default when skipping
		return nil
	}

	_, _ = fmt.Println()
	_, _ = fmt.Println(Colorize(Bold, "=== Optional Dependencies ==="))
	_, _ = fmt.Println()

	for i, dep := range deps {
		desc := ""
		if dep.Description != "" {
			desc = Colorize(Cyan, " - "+dep.Description)
		}
		fmt.Printf("  %d) %s%s\n", i+1, dep.Name, desc)
	}
	fmt.Println()

	fmt.Print(Colorize(Bold, "Enter numbers to install (e.g., 1,3,5), 'all', or press Enter to skip: "))

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil
	}

	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return nil
	}

	if input == "all" {
		selected := make([]string, len(deps))
		for i, dep := range deps {
			selected[i] = dep.Name
		}
		return selected
	}

	// Parse comma-separated numbers
	selected := make([]string, 0)
	parts := strings.Split(input, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		var num int
		if _, err := fmt.Sscanf(part, "%d", &num); err == nil {
			if num >= 1 && num <= len(deps) {
				selected = append(selected, deps[num-1].Name)
			}
		}
	}

	return selected
}

// PrintReinstallList prints packages to be reinstalled
func PrintReinstallList(packages []string) {
	PrintPackageList("Packages to reinstall", packages, Magenta)
}
