package cmd

import (
	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/output"
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
	Run: func(cmd *cobra.Command, args []string) {
		eng, err := NewEngineWithConfig(nil)
		if err != nil {
			output.Error("Failed to initialize engine: %v", err)
			return
		}

		if !markAsDeps && !markAsExplicit {
			output.Error("Must specify either --as-deps or --as-explicit")
			return
		}
		if markAsDeps && markAsExplicit {
			output.Error("Cannot specify both --as-deps and --as-explicit")
			return
		}

		pkgName := args[0]
		var reason core.InstallReason
		var reasonStr string

		if markAsDeps {
			reason = core.InstallReasonDependency
			reasonStr = "dependency"
		} else {
			reason = core.InstallReasonExplicit
			reasonStr = "explicit"
		}

		output.Info("Marking %s as %s...", pkgName, reasonStr)
		if err := eng.SetInstallReason(pkgName, reason); err != nil {
			output.Error("Failed to mark package: %v", err)
			return
		}
		output.Success("Package marked successfully")
	},
}

func init() {
	MarkCmd.Flags().BoolVar(&markAsDeps, "as-deps", false, "Mark as installed as a dependency")
	MarkCmd.Flags().BoolVar(&markAsExplicit, "as-explicit", false, "Mark as explicitly installed")
}
