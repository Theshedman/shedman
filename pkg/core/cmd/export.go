package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/pkg/core"
)

var ExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export list of explicitly installed packages",
	Long: `Export a list of all explicitly installed packages to stdout.
This list can be saved to a file and used with 'shedman import' to restore them.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, err := NewEngineWithConfig(nil) // Use factory default
		if err != nil {
			return err
		}
		return RunExport(eng, cmd.OutOrStdout())
	},
}

// RunExport exports installed packages to a file
func RunExport(eng *core.Engine, w io.Writer) error {
	pkgs, err := eng.ListExplicitPackages()
	if err != nil {
		return fmt.Errorf("failed to list packages: %w", err)
	}

	for _, pkg := range pkgs {
		fmt.Fprintln(w, pkg)
	}

	return nil
}
