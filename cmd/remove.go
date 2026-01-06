package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/pkg/shedman/backend"
	"github.com/theshedman/shedman/pkg/shedman/backend/pacman"
	"github.com/theshedman/shedman/pkg/shedman/output"
)

var (
	removeRecursive bool
	removePurge     bool
)

var removeCmd = &cobra.Command{
	Use:   "remove [packages...]",
	Short: "Remove packages",
	Long: `Remove installed packages.

Examples:
  shedman remove neovim           # Remove package
  shedman remove neovim --purge   # Remove + delete configs
  shedman remove neovim -s        # Remove + orphan deps`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Dry-run mode
		if dryRunFlag {
			cmd.Println("Dry-run mode: would remove the following packages:")
			for _, pkg := range args {
				cmd.Printf("  - %s\n", pkg)
			}
			if removeRecursive {
				cmd.Println("  (with --recursive: would also remove unused dependencies)")
			}
			if removePurge {
				cmd.Println("  (with --purge: would also delete configuration files)")
			}
			return nil
		}

		// Get pacman backend
		pacmanBackend, err := pacman.New()
		if err != nil {
			return fmt.Errorf("pacman backend not available: %w", err)
		}

		// Verify packages are installed
		var notInstalled []string
		var toRemove []string
		for _, pkg := range args {
			if !pacmanBackend.IsInstalled(pkg) {
				notInstalled = append(notInstalled, pkg)
			} else {
				toRemove = append(toRemove, pkg)
			}
		}

		if len(notInstalled) > 0 {
			for _, pkg := range notInstalled {
				output.Warning("Package not installed: %s", pkg)
			}
		}

		if len(toRemove) == 0 {
			return fmt.Errorf("no packages to remove")
		}

		// Show what we're removing
		if !quietFlag {
			output.Info("Removing %d package(s)...", len(toRemove))
			for _, pkg := range toRemove {
				fmt.Printf("  → %s\n", pkg)
			}
		}

		// Build remove options
		opts := backend.RemoveOptions{
			Cascade:   false, // We use recursive instead
			NoSave:    removePurge,
			Recursive: removeRecursive,
			NoConfirm: yesFlag,
		}

		// Execute removal
		if err := pacmanBackend.Remove(toRemove, opts); err != nil {
			return fmt.Errorf("removal failed: %w", err)
		}

		if !quietFlag {
			output.Success("Removal complete.")
		}

		return nil
	},
}

// GetRemoveCmd returns the remove command for testing
func GetRemoveCmd() *cobra.Command {
	return removeCmd
}

func init() {
	removeCmd.Flags().BoolVarP(&removeRecursive, "recursive", "s", false, "Remove unused dependencies")
	removeCmd.Flags().BoolVar(&removePurge, "purge", false, "Also remove configuration files")

	rootCmd.AddCommand(removeCmd)
}
