package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, err := NewEngineWithConfig(nil)
		if err != nil {
			return fmt.Errorf("failed to initialize engine: %w", err)
		}

		if err := RunOrphans(eng, cmd.OutOrStdout(), removeOrphans); err != nil {
			return fmt.Errorf("orphans operation failed: %w", err)
		}
		return nil
	},
}

func init() {
	OrphansCmd.Flags().BoolVar(&removeOrphans, "remove", false, "Remove found orphans")
}

func RunOrphans(eng *core.Engine, w io.Writer, remove bool) error {
	fmt.Fprintln(w, "Searching for orphans...")
	orphans, err := eng.ListOrphans()
	if err != nil {
		return fmt.Errorf("failed to list orphans: %w", err)
	}

	if len(orphans) == 0 {
		_, _ = fmt.Fprintln(w, "No orphans found.")

		return nil
	}

	if remove {
		_, _ = fmt.Fprintf(w, "Found %d orphans: %s\n", len(orphans), strings.Join(orphans, " "))

		if err := eng.RemoveOrphans(orphans); err != nil {
			return fmt.Errorf("removal failed: %w", err)
		}
		fmt.Fprintln(w, "Orphans removed.")
	} else {
		for _, pkg := range orphans {
			_, _ = fmt.Fprintln(w, pkg)

		}
		_, _ = fmt.Fprintln(w, "\nUse --remove to uninstall them.")

	}
	return nil
}
