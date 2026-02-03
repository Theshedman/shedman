package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/internal/util"
	"github.com/theshedman/shedman/pkg/core"
)

var (
	buildClean     bool
	buildInstall   bool
	buildNoConfirm bool
	buildSynDeps   bool
	buildEdit      bool
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

		cfg, err := config.LoadDefault()
		if err != nil {
			cfg = config.Default()
		}

		eng, err := NewEngineWithConfig(cfg)
		if err != nil {
			return fmt.Errorf("failed to initialize engine: %w", err)
		}

		if buildEdit {
			if err := editPKGBUILD(dir, resolveEditor(cfg)); err != nil {
				return err
			}
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
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Build completed successfully.")

		return nil
	},
}

func init() {
	BuildCmd.Flags().BoolVarP(&buildClean, "clean", "c", false, "Clean up work files after build")
	BuildCmd.Flags().BoolVarP(&buildInstall, "install", "i", false, "Install package after successful build")
	BuildCmd.Flags().BoolVarP(&buildSynDeps, "syncdeps", "s", true, "Install missing dependencies with pacman")
	BuildCmd.Flags().BoolVar(&buildNoConfirm, "noconfirm", false, "Do not ask for confirmation")
	BuildCmd.Flags().BoolVar(&buildEdit, "edit", false, "Edit PKGBUILD before build")
}

func RunBuild(eng *core.Engine, w io.Writer, dir string, opts core.BuildOptions) error {
	_, _ = fmt.Fprintf(w, "Building package in %s...\n", dir)

	return eng.Build(dir, opts)
}

var editorRunner = func(editor string, args []string) error {
	cmd := exec.Command(editor, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// editorValidator validates the editor path. Can be replaced in tests.
var editorValidator = util.ValidateExecutablePath

func editPKGBUILD(dir, editor string) error {
	pkgbuild := filepath.Join(dir, "PKGBUILD")
	if _, err := os.Stat(pkgbuild); err != nil {
		return fmt.Errorf("PKGBUILD not found in %s", dir)
	}

	if editor == "" {
		editor = "vim"
	}

	// Validate the editor path to prevent command injection
	if err := editorValidator(editor); err != nil {
		return fmt.Errorf("invalid editor: %w", err)
	}

	return editorRunner(editor, []string{pkgbuild})
}

func resolveEditor(cfg *config.Config) string {
	if cfg != nil && cfg.General.Editor != "" {
		return cfg.General.Editor
	}
	if env := os.Getenv("EDITOR"); env != "" {
		return env
	}
	return "vim"
}
