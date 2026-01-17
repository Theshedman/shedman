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
		cfg, err := config.LoadDefault()
		if err != nil {
			output.Warning("Failed to load config, using defaults: %v", err)
			cfg = config.Default()
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		if dryRun {
			return handleRemoveDryRun(cmd.OutOrStdout(), args)
		}

		eng, err := NewEngineWithConfig(cfg)
		if err != nil {
			return fmt.Errorf("failed to initialize engine: %w", err)
		}

		// Filter ignored packages based on config
		// To match test simplicity, we filter here or pass clean list.
		cleanArgs := filterIgnoredPackages(args, cfg.Packages.IgnorePkg)

		if len(cleanArgs) == 0 {
			return fmt.Errorf("no packages to remove (check ignore list)")
		}

		quiet, _ := cmd.Flags().GetBool("quiet")
		yes, _ := cmd.Flags().GetBool("yes")

		noSave := removePurge || removeNosave

		opts := core.RemoveOptions{
			Cascade:   removeCascade,
			NoSave:    noSave,
			Recursive: removeRecursive,
			NoConfirm: yes || !cfg.General.Confirm,
		}

		// Inject Stdout/Stderr via Cobra
		if quiet {
			return RunRemove(eng, io.Discard, cleanArgs, opts)
		}
		return RunRemove(eng, cmd.OutOrStdout(), cleanArgs, opts)
	},
}

func RunRemove(eng *core.Engine, w io.Writer, pkgs []string, opts core.RemoveOptions) error {

	backend := eng.GetOfficialBackend()
	if backend == nil {
		return core.ErrBackendNotFound
	}

	var toRemove []string
	for _, p := range pkgs {
		if backend.IsInstalled(p) {
			toRemove = append(toRemove, p)
		} else {
			_, _ = fmt.Fprintf(w, "Warning: Package %s is not installed\n", p)

		}
	}

	if len(toRemove) == 0 {
		return fmt.Errorf("no packages to remove")
	}

	_, _ = fmt.Fprintf(w, "Removing %d official package(s)...\n", len(toRemove))

	if !opts.NoConfirm {
		_, _ = fmt.Fprintf(w, "The following packages will be removed:\n")

		for _, p := range toRemove {
			_, _ = fmt.Fprintf(w, "  -> %s\n", p)

		}

		if !opts.NoConfirm && eng.GetConfig().General.Confirm {
			if !output.Confirm("Proceed with removal?", output.ConfirmOptions{Default: true}) {
				return fmt.Errorf("removal cancelled")
			}
		}
	}

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
	_, _ = fmt.Fprintf(w, "Dry-run mode (backend: %s):\n", backendName)

	fmt.Fprintln(w, "Would remove the following packages:")
	for _, pkg := range args {
		_, _ = fmt.Fprintf(w, "  - %s\n", pkg)

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
