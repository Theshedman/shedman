package cmd

import (
	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/output"
	"github.com/theshedman/shedman/pkg/core"
)

// DownloadCmd represents the download command
var DownloadCmd = &cobra.Command{
	Use:   "download [packages...]",
	Short: "Download packages without installing",
	Long:  `Downloads the specified packages to the cache without installing them.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		eng, err := NewEngineWithConfig(nil)
		if err != nil {
			output.Error("Failed to initialize engine: %v", err)
			return
		}

		if err := RunDownload(eng, args); err != nil {
			output.Error("%v", err)
		} else {
			output.Success("Downloaded packages successfully.")
		}
	},
}

// RunDownload executes the download logic
func RunDownload(eng *core.Engine, pkgs []string) error {
	output.Info("Downloading packages...")
	options := core.InstallOptions{
		DownloadOnly: true,
		// Standard install options apply
	}
	return eng.Install(pkgs, options)
}
