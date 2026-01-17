package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/pkg/core"
)

var (
	markAsDeps     bool
	markAsExplicit bool
)

// MarkCmd represents the mark command
var MarkCmd = &cobra.Command{
	Use:   "mark [package]",
	Short: "Update package install reason",
	Long: `Mark a package as installed as a dependency or explicitly installed.
Example:
  shedman mark neovim --as-explicit
  shedman mark libgit2 --as-deps`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, err := NewEngineWithConfig(nil)
		if err != nil {
			return fmt.Errorf("failed to initialize engine: %w", err)
		}

		if !markAsDeps && !markAsExplicit {
			return fmt.Errorf("must specify either --as-deps or --as-explicit")
		}
		if markAsDeps && markAsExplicit {
			return fmt.Errorf("cannot specify both --as-deps and --as-explicit")
		}

		pkgName := args[0]

		if err := RunMark(eng, cmd.OutOrStdout(), pkgName, markAsDeps, markAsExplicit); err != nil {
			return err
		}
		return nil
	},
}

func RunMark(eng *core.Engine, w io.Writer, pkgName string, asDeps, asExplicit bool) error {
	var reason core.InstallReason
	var reasonStr string

	if asDeps {
		reason = core.InstallReasonDependency
		reasonStr = "dependency"
	} else {
		// asExplicit is implied true if asDeps is false based on caller check
		reason = core.InstallReasonExplicit
		reasonStr = "explicit"
	}

	_, _ = fmt.Fprintf(w, "Marking %s as %s...\n", pkgName, reasonStr)

	if err := eng.SetInstallReason(pkgName, reason); err != nil {
		return fmt.Errorf("failed to mark package: %w", err)
	}
	_, _ = fmt.Fprintln(w, "Package marked successfully")

	return nil
}

func init() {
	MarkCmd.Flags().BoolVar(&markAsDeps, "as-deps", false, "Mark as installed as a dependency")
	MarkCmd.Flags().BoolVar(&markAsExplicit, "as-explicit", false, "Mark as explicitly installed")
}
