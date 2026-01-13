package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/output"
	"github.com/theshedman/shedman/pkg/core"
)

var (
	buildClean     bool
	buildInstall   bool
	buildNoConfirm bool
	buildSynDeps   bool
)

// BuildCmd represents the build command
var BuildCmd = &cobra.Command{
	Use:   "build [directory]",
	Short: "Build package from PKGBUILD",
	Long:  `Build a package from a PKGBUILD file in the specified directory (defaulting to current directory). Wraps makepkg.`,
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}

		// Ensure we are in the directory if it's not .
		// Use absolute path

		eng, err := NewEngineWithConfig(nil)
		if err != nil {
			output.Error("Failed to initialize engine: %v", err)
			return
		}

		// Default options for makepkg usually include -s (syncdeps)
		// We expose explicit flags override.
		opts := core.BuildOptions{
			Clean:     buildClean,
			Install:   buildInstall,
			NoConfirm: buildNoConfirm,
			SynDeps:   buildSynDeps,
		}

		// If user didn't specify anything, rely on flag defaults
		// makepkg -si is common. Let's rely on flags.
		// If flags are all false, we just run makepkg (build only).

		if err := RunBuild(eng, dir, opts); err != nil {
			output.Error("Build failed: %v", err)
			os.Exit(1)
		}
		output.Success("Build completed successfully.")
	},
}

func init() {
	BuildCmd.Flags().BoolVarP(&buildClean, "clean", "c", false, "Clean up work files after build")
	BuildCmd.Flags().BoolVarP(&buildInstall, "install", "i", false, "Install package after successful build")
	BuildCmd.Flags().BoolVarP(&buildSynDeps, "syncdeps", "s", true, "Install missing dependencies with pacman")
	BuildCmd.Flags().BoolVar(&buildNoConfirm, "noconfirm", false, "Do not ask for confirmation")
}

// RunBuild executes the build logic
func RunBuild(eng *core.Engine, dir string, opts core.BuildOptions) error {
	output.Info("Building package in %s...", dir)
	return eng.Build(dir, opts)
}
