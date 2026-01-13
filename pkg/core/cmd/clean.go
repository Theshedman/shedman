package cmd

import (
	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/output"
	"github.com/theshedman/shedman/pkg/core"
)

var (
	cleanAll  bool
	cleanKeep int
)

// CleanCmd represents the clean command
var CleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean the package cache",
	Long:  `Remove uninstalled or all packages from the cache to free up disk space. Use --keep to retain recent versions.`,
	Run: func(cmd *cobra.Command, args []string) {
		eng, err := NewEngineWithConfig(nil)
		if err != nil {
			output.Error("Failed to initialize engine: %v", err)
			return
		}

		if err := RunClean(eng, cleanAll, cleanKeep); err != nil {
			output.Error("Clean failed: %v", err)
			return
		}
		output.Success("Cache cleaned successfully.")
	},
}

func init() {
	CleanCmd.Flags().BoolVar(&cleanAll, "all", false, "Remove all files from cache (not just uninstalled)")
	CleanCmd.Flags().IntVar(&cleanKeep, "keep", 0, "Number of recent versions to keep (implies using paccache)")
}

// RunClean executes the clean logic
func RunClean(eng *core.Engine, all bool, keep int) error {
	output.Info("Cleaning package cache...")
	opts := core.CleanOptions{
		All:  all,
		Keep: keep,
	}
	return eng.CleanCache(opts)
}
