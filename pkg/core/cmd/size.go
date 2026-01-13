package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/output"
	"github.com/theshedman/shedman/pkg/core"
)

// SizeCmd represents the size command
var SizeCmd = &cobra.Command{
	Use:   "size [package]",
	Short: "Show package size",
	Long:  `Show the installed size and download size of a package.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		eng, err := NewEngineWithConfig(nil)
		if err != nil {
			output.Error("Failed to initialize engine: %v", err)
			return
		}

		if err := RunSize(eng, args[0]); err != nil {
			output.Error("%v", err)
		}
	},
}

// RunSize executes the size logic
func RunSize(eng *core.Engine, pkg string) error {
	info, err := eng.Info(pkg)
	if err != nil {
		return fmt.Errorf("failed to get package info: %w", err)
	}

	output.Info("Package: %s", info.Name)
	fmt.Printf("  Installed Size: %s\n", formatBytes(info.InstalledSize))
	fmt.Printf("  Download Size:  %s\n", formatBytes(info.Size))

	return nil
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
