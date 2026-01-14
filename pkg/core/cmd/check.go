package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/pkg/core"
)

// CheckCmd represents the check command
var CheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check package database consistency",
	Long: `Check local package database for internal consistency (wraps pacman -Dk).
	
Example:
  shedman check`,
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, err := NewEngineWithConfig(nil)
		if err != nil {
			return fmt.Errorf("failed to initialize engine: %w", err)
		}

		return RunCheck(eng, cmd.OutOrStdout())
	},
}

func init() {
	// No specific flags for check command yet
}

func RunCheck(eng *core.Engine, w io.Writer) error {
	fmt.Fprintln(w, "Checking package database consistency...")
	if err := eng.CheckDatabase(); err != nil {
		return fmt.Errorf("database check failed: %w", err)
	}
	fmt.Fprintln(w, "Package database is consistent")
	return nil
}
