package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, err := NewEngineWithConfig(nil)
		if err != nil {
			return fmt.Errorf("failed to initialize engine: %w", err)
		}

		if err := RunClean(eng, cmd.OutOrStdout(), cleanAll, cleanKeep); err != nil {
			// output.Error handles printing but we return error here for RunE
			return fmt.Errorf("clean failed: %w", err)
		}

		fmt.Println("Cache cleaned successfully.")
		// But RunClean writes to 'w'. If success message is specific to CLI, we print it here.
		return nil
	},
}

func init() {
	CleanCmd.Flags().BoolVar(&cleanAll, "all", false, "Remove all files from cache (not just uninstalled)")
	CleanCmd.Flags().IntVar(&cleanKeep, "keep", 0, "Number of recent versions to keep (implies using paccache)")
}

func RunClean(eng *core.Engine, w io.Writer, all bool, keep int) error {
	fmt.Fprintln(w, "Cleaning package cache...")
	opts := core.CleanOptions{
		All:  all,
		Keep: keep,
	}
	return eng.CleanCache(opts)
}
