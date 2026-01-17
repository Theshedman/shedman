package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/output"
	"github.com/theshedman/shedman/pkg/core"
)

var repairLock bool

// RepairCmd represents the repair command
var RepairCmd = &cobra.Command{
	Use:   "repair",
	Short: "Repair internal database issues",
	Long:  `Repair common package manager issues, such as removing stale lock files.`,
	Run: func(cmd *cobra.Command, args []string) {
		eng, err := NewEngineWithConfig(nil)
		if err != nil {
			output.Error("Failed to initialize engine: %v", err)
			return
		}

		if repairLock {
			if err := RunRepair(eng, cmd.OutOrStdout(), "lock"); err != nil {
				output.Error("Failed to remove lock: %v", err)
			} else {
				output.Success("Lock file removed.")
			}
			return
		}

		// If no specific repair action, show help
		_ = cmd.Help()

	},
}

func init() {
	RepairCmd.Flags().BoolVar(&repairLock, "lock", false, "Remove stale pacman lock file")
}

// RunRepair executes repair actions
func RunRepair(eng *core.Engine, w io.Writer, action string) error {
	switch action {
	case "lock":
		_, _ = fmt.Fprintln(w, "Removing stale lock file...")

		return eng.RepairLock()
	default:
		return fmt.Errorf("unknown repair action: %s", action)
	}
}
