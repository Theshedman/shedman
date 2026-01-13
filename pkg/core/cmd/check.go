package cmd

import (
	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/output"
)

// CheckCmd represents the check command
var CheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check package database consistency",
	Long: `Check local package database for internal consistency (wraps pacman -Dk).
	
Example:
  shedman check`,
	Run: func(cmd *cobra.Command, args []string) {
		eng, err := NewEngineWithConfig(nil)
		if err != nil {
			output.Error("Failed to initialize engine: %v", err)
			return
		}

		output.Info("Checking package database consistency...")
		if err := eng.CheckDatabase(); err != nil {
			output.Error("Database check failed: %v", err)
			// Don't return error to cobra to avoid duplicate usage printing,
			// but we should exit non-zero if real app.
			// Cobra handles Run errors by printing usage. We prefer standardized output.
			return
		}
		output.Success("Package database is consistent")
	},
}

func init() {
	// No specific flags for check command yet
}
