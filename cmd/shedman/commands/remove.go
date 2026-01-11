package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/internal/output"
	"github.com/theshedman/shedman/pkg/backend"
	"github.com/theshedman/shedman/pkg/core/installer"
	"github.com/theshedman/shedman/pkg/core/resolver"
)

var (
	removeRecursive bool
	removePurge     bool
	removeCascade   bool
	removeNosave    bool
)

// ShedInstalledProvider adapts ShedInstaller to resolver.InstalledProvider
type ShedInstalledProvider struct {
	installer *installer.ShedInstaller
}

func (s ShedInstalledProvider) GetInstalledPackages() []resolver.InstalledPackage {
	var result []resolver.InstalledPackage

	names, _ := s.installer.ListInstalled()
	for _, name := range names {
		// Manually read manifest to get dependencies
		installedDir := fmt.Sprintf("%s/%s", s.installer.GetInstalledDir(), name)
		manifest, err := s.installer.ReadManifest(installedDir)
		if err != nil {
			continue
		}

		result = append(result, resolver.InstalledPackage{
			Name:    manifest.Name,
			Depends: manifest.Depends,
		})
	}
	return result
}

var removeCmd = &cobra.Command{
	Use:   "remove [packages...]",
	Short: "Remove packages",
	Long: `Remove installed packages.

Supports multiple package sources across different Linux distributions:
  - Official packages (via distro's native package manager)
  - .shed packages (native shedman format)

The appropriate backend is auto-detected based on your distribution:
  - Arch/Manjaro/EndeavourOS → pacman
  - Debian/Ubuntu/Mint → apt
  - Fedora/CentOS/RHEL → dnf
  - openSUSE → zypper

Examples:
  shedman remove neovim           # Remove package
  shedman remove neovim --purge   # Remove + delete configs
  shedman remove neovim -s        # Remove + orphan deps
  shedman remove neovim --cascade # Remove + cascade to dependents`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration
		cfg, err := config.LoadDefault()
		if err != nil {
			output.Warning("Failed to load config, using defaults: %v", err)
			cfg = config.Default()
		}

		// Dry-run mode
		if dryRunFlag {
			return handleRemoveDryRun(cmd, args, cfg)
		}

		// Get the appropriate official backend for this distro
		officialBackend, err := backend.DetectBackendWithConfig(&cfg.Backend)
		if err != nil {
			output.Warning("Official backend not available: %v", err)
			officialBackend = nil
		}

		// Group packages by backend (official vs .shed)
		officialPkgs, shedPkgs, notFound, err := categorizePackagesForRemoval(args, cfg, officialBackend)
		if err != nil {
			return err
		}

		// Report not found packages
		for _, pkg := range notFound {
			output.Warning("Package not installed: %s", pkg)
		}

		// Handle Recursive Removal for .shed packages
		if removeRecursive && len(shedPkgs) > 0 {
			shedInstaller := installer.NewShedInstaller()
			provider := ShedInstalledProvider{installer: shedInstaller}

			// Calculate removal list including orphans
			expandedList := resolver.CalculateRecursiveRemoval(shedPkgs, provider)

			// Identify newly added packages for user info
			originalSet := make(map[string]bool)
			for _, p := range shedPkgs {
				originalSet[p] = true
			}

			var added []string
			for _, p := range expandedList {
				if !originalSet[p] {
					added = append(added, p)
				}
			}

			if len(added) > 0 {
				output.Info("Recursive removal added: %s", strings.Join(added, ", "))
				shedPkgs = expandedList
			}
		}

		totalToRemove := len(officialPkgs) + len(shedPkgs)
		if totalToRemove == 0 {
			return fmt.Errorf("no packages to remove")
		}

		// Show what we're removing
		if !quietFlag {
			backendName := "unknown"
			if officialBackend != nil {
				backendName = officialBackend.Name()
			}
			output.Info("Packages to remove (%d):", totalToRemove)
			for _, pkg := range officialPkgs {
				fmt.Printf("  → %s [%s]\n", pkg, backendName)
			}
			for _, pkg := range shedPkgs {
				fmt.Printf("  → %s [shed]\n", pkg)
			}
			fmt.Println()
		}

		// Confirmation prompt (unless --yes or config.General.Confirm is false)
		if !yesFlag && cfg.General.Confirm {
			prompt := fmt.Sprintf("Remove %d package(s)?", totalToRemove)
			if !output.Confirm(prompt, output.ConfirmOptions{Default: false}) {
				output.Info("Removal cancelled.")
				return nil
			}
		}

		// Execute removal for each backend
		if err := executeRemoval(officialBackend, officialPkgs, shedPkgs, cfg); err != nil {
			return err
		}

		if !quietFlag {
			output.Success("Removal complete.")
		}

		return nil
	},
}

// categorizePackagesForRemoval groups packages by their backend (official vs shed)
func categorizePackagesForRemoval(args []string, cfg *config.Config, officialBackend backend.OfficialBackend) (officialPkgs, shedPkgs, notFound []string, err error) {
	// Build ignore set from config
	ignoreSet := make(map[string]bool)
	for _, pkg := range cfg.Packages.IgnorePkg {
		ignoreSet[pkg] = true
	}

	// Initialize ShedInstaller to check for .shed packages
	shedInstaller := installer.NewShedInstaller()

	for _, pkg := range args {
		// Check if package is ignored
		if ignoreSet[pkg] {
			output.Warning("Package %s is in IgnorePkg list, skipping", pkg)
			continue
		}

		// Check if it's a .shed package first (takes priority)
		if shedInstaller.IsInstalled(pkg) {
			shedPkgs = append(shedPkgs, pkg)
			continue
		}

		// Check if it's installed via official backend
		if officialBackend != nil && officialBackend.IsInstalled(pkg) {
			officialPkgs = append(officialPkgs, pkg)
			continue
		}

		// Package not found in any backend
		notFound = append(notFound, pkg)
	}

	return officialPkgs, shedPkgs, notFound, nil
}

// executeRemoval removes packages using the appropriate backend
func executeRemoval(officialBackend backend.OfficialBackend, officialPkgs, shedPkgs []string, cfg *config.Config) error {
	// Remove official packages via detected backend
	if len(officialPkgs) > 0 {
		if officialBackend == nil {
			return fmt.Errorf("official backend not available, cannot remove: %v", officialPkgs)
		}

		// Merge --purge and --nosave
		noSave := removePurge || removeNosave

		opts := backend.RemoveOptions{
			Cascade:   removeCascade,
			NoSave:    noSave,
			Recursive: removeRecursive,
			NoConfirm: yesFlag || !cfg.General.Confirm,
		}

		if !quietFlag {
			output.Info("Removing %d %s package(s)...", len(officialPkgs), officialBackend.Name())
		}

		if err := officialBackend.Remove(officialPkgs, opts); err != nil {
			return fmt.Errorf("%s removal failed: %w", officialBackend.Name(), err)
		}
	}

	// Remove .shed packages
	if len(shedPkgs) > 0 {
		shedInstaller := installer.NewShedInstaller()

		if !quietFlag {
			output.Info("Removing %d .shed package(s)...", len(shedPkgs))
		}

		for _, pkg := range shedPkgs {
			if verboseFlag {
				output.Info("Removing .shed package: %s", pkg)
			}
			if err := shedInstaller.Remove(pkg); err != nil {
				return fmt.Errorf("failed to remove .shed package %s: %w", pkg, err)
			}
		}
	}

	return nil
}

// handleRemoveDryRun shows what would be removed without actually removing
func handleRemoveDryRun(cmd *cobra.Command, args []string, cfg *config.Config) error {
	// Detect backend for display
	backendName := backend.GetBackendName()

	cmd.Printf("Dry-run mode (backend: %s):\n", backendName)
	cmd.Println("Would remove the following packages:")
	for _, pkg := range args {
		cmd.Printf("  - %s\n", pkg)
	}
	if removeRecursive {
		cmd.Println("  (with --recursive: would also remove unused dependencies)")
	}
	if removeCascade {
		cmd.Println("  (with --cascade: would also remove packages depending on these)")
	}
	if removePurge || removeNosave {
		cmd.Println("  (with --purge/--nosave: would also delete configuration files)")
	}
	return nil
}

// GetRemoveCmd returns the remove command for testing
func GetRemoveCmd() *cobra.Command {
	return removeCmd
}

func init() {
	removeCmd.Flags().BoolVarP(&removeRecursive, "recursive", "s", false, "Remove unused dependencies")
	removeCmd.Flags().BoolVar(&removeCascade, "cascade", false, "Remove packages that depend on these")
	removeCmd.Flags().BoolVar(&removePurge, "purge", false, "Also remove configuration files")
	removeCmd.Flags().BoolVar(&removeNosave, "nosave", false, "Don't save configuration files (alias for --purge)")

	rootCmd.AddCommand(removeCmd)
}
