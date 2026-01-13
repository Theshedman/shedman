package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/output"
	"github.com/theshedman/shedman/pkg/core"
)

var (
	removeOrphans bool
)

// OrphansCmd represents the orphans command
var OrphansCmd = &cobra.Command{
	Use:   "orphans",
	Short: "Manage orphaned packages (unused dependencies)",
	Long:  `List or remove packages that were installed as dependencies but are no longer required by any installed package.`,
	Run: func(cmd *cobra.Command, args []string) {
		eng, err := NewEngineWithConfig(nil)
		if err != nil {
			output.Error("Failed to initialize engine: %v", err)
			return
		}

		if err := RunOrphans(eng, removeOrphans); err != nil {
			output.Error("Orphans operation failed: %v", err)
			return
		}
	},
}

func init() {
	OrphansCmd.Flags().BoolVar(&removeOrphans, "remove", false, "Remove found orphans")
}

// RunOrphans executes the orphans logic
func RunOrphans(eng *core.Engine, remove bool) error {
	output.Info("Searching for orphans...")
	orphans, err := eng.ListOrphans()
	if err != nil {
		return fmt.Errorf("failed to list orphans: %w", err)
	}

	if len(orphans) == 0 {
		output.Success("No orphans found.")
		return nil
	}

	if remove {
		output.Info("Found %d orphans: %s", len(orphans), strings.Join(orphans, " "))
		if err := eng.RemoveOrphans(orphans); err != nil {
			return fmt.Errorf("removal failed: %w", err)
		}
		output.Success("Orphans removed.")
	} else {
		for _, pkg := range orphans {
			fmt.Println(pkg)
		}
		output.Info("\nUse --remove to uninstall them.")
	}
	return nil
}
