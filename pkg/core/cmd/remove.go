package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/internal/output"
	"github.com/theshedman/shedman/pkg/core"
)

var (
	removeRecursive bool
	removePurge     bool
	removeCascade   bool
	removeNosave    bool
)

var RemoveCmd = &cobra.Command{
	Use:   "remove [packages...]",
	Short: "Remove packages",
	Long: `Remove installed packages.
	
Supports:
  - Official packages (via pacman)

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

		// Dry-run handle
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		if dryRun {
			return handleRemoveDryRun(cmd.OutOrStdout(), args)
		}

		eng, err := NewEngineWithConfig(cfg)
		if err != nil {
			return fmt.Errorf("failed to initialize engine: %w", err)
		}

		// Filter ignored packages based on config
		// This logic stays in Command layer or passed to RunRemove?
		// To match test simplicity, we filter here or pass clean list.
		// Let's filter explicitly here to keep RunRemove pure.
		cleanArgs := filterIgnoredPackages(args, cfg.Packages.IgnorePkg)

		if len(cleanArgs) == 0 {
			return fmt.Errorf("no packages to remove (check ignore list)")
		}

		quiet, _ := cmd.Flags().GetBool("quiet")
		yes, _ := cmd.Flags().GetBool("yes")

		// Merge --purge and --nosave
		noSave := removePurge || removeNosave

		opts := core.RemoveOptions{
			Cascade:   removeCascade,
			NoSave:    noSave,
			Recursive: removeRecursive,
			NoConfirm: yes || !cfg.General.Confirm,
		}

		// Inject Stdout/Stderr via Cobra
		// Note: RunRemove uses injected Writer for primary output
		if quiet {
			return RunRemove(eng, io.Discard, cleanArgs, opts)
		}
		return RunRemove(eng, cmd.OutOrStdout(), cleanArgs, opts)
	},
}

// RunRemove executes the removal logic
// Refactored for TDD: Logic isolated from CLI/Config loading
func RunRemove(eng *core.Engine, w io.Writer, pkgs []string, opts core.RemoveOptions) error {
	// 1. Verify availability/installed status using Engine
	// Since generic Engine doesn't have "IsInstalled" readily exposed for batch without backend,
	// we will iterate or trust the engine's Remove to error out?
	// The test expects us to verify installation.
	// We'll use GetOfficialBackend for verification if available, as 'remove' targets official/local pkgs.

	backend := eng.GetOfficialBackend()
	if backend == nil {
		return core.ErrBackendNotFound
	}

	var toRemove []string
	for _, p := range pkgs {
		if backend.IsInstalled(p) {
			toRemove = append(toRemove, p)
		} else {
			// Warn but don't fail entire batch immediately?
			// Standard behavior: warn about missing.
			// Since we don't have a logger injected besides 'w', we print to 'w'.
			fmt.Fprintf(w, "Warning: package '%s' is not installed\n", p)
		}
	}

	if len(toRemove) == 0 {
		return fmt.Errorf("no packages to remove")
	}

	// 2. Output plan
	fmt.Fprintf(w, "Removing %d official package(s)...\n", len(toRemove))
	for _, p := range toRemove {
		fmt.Fprintf(w, "  -> %s\n", p)
	}

	// 3. Confirm (ShedMan Level) - Skipped if NoConfirm
	// If NoConfirm is false, we technically should prompt.
	// But since this function doesn't take a Reader, we assume caller did verification
	// OR we trust opts.NoConfirm passed down to backend.
	// The CLI layer handles the interactive "Are you sure?".
	// Here we proceed to call backend.

	// 4. Create Engine options and Execute
	// We pass the filtered list
	return eng.Remove(toRemove, opts)
}

func filterIgnoredPackages(args []string, ignored []string) []string {
	ignoreSet := make(map[string]bool)
	for _, p := range ignored {
		ignoreSet[p] = true
	}
	var clean []string
	for _, p := range args {
		if ignoreSet[p] {
			output.Warning("Skipping ignored package: %s", p)
			continue
		}
		clean = append(clean, p)
	}
	return clean
}

func handleRemoveDryRun(w io.Writer, args []string) error {
	backendName := core.GetBackendName()
	fmt.Fprintf(w, "Dry-run mode (backend: %s):\n", backendName)
	fmt.Fprintln(w, "Would remove the following packages:")
	for _, pkg := range args {
		fmt.Fprintf(w, "  - %s\n", pkg)
	}
	// ... details
	return nil
}

func init() {
	RemoveCmd.Flags().BoolVarP(&removeRecursive, "recursive", "s", false, "Remove unused dependencies")
	RemoveCmd.Flags().BoolVar(&removeCascade, "cascade", false, "Remove packages that depend on these")
	RemoveCmd.Flags().BoolVar(&removePurge, "purge", false, "Also remove configuration files")
	RemoveCmd.Flags().BoolVar(&removeNosave, "nosave", false, "Don't save configuration files (alias for --purge)")
}
