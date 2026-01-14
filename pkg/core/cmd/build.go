package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}

		eng, err := NewEngineWithConfig(nil)
		if err != nil {
			return fmt.Errorf("failed to initialize engine: %w", err)
		}

		// Default options for makepkg
		opts := core.BuildOptions{
			Clean:     buildClean,
			Install:   buildInstall,
			NoConfirm: buildNoConfirm,
			SynDeps:   buildSynDeps,
		}

		if err := RunBuild(eng, cmd.OutOrStdout(), dir, opts); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Build completed successfully.")
		return nil
	},
}

func init() {
	BuildCmd.Flags().BoolVarP(&buildClean, "clean", "c", false, "Clean up work files after build")
	BuildCmd.Flags().BoolVarP(&buildInstall, "install", "i", false, "Install package after successful build")
	BuildCmd.Flags().BoolVarP(&buildSynDeps, "syncdeps", "s", true, "Install missing dependencies with pacman")
	BuildCmd.Flags().BoolVar(&buildNoConfirm, "noconfirm", false, "Do not ask for confirmation")
}

func RunBuild(eng *core.Engine, w io.Writer, dir string, opts core.BuildOptions) error {
	fmt.Fprintf(w, "Building package in %s...\n", dir)
	return eng.Build(dir, opts)
}
