package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/pkg/core"
)

// DownloadCmd represents the download command
var DownloadCmd = &cobra.Command{
	Use:   "download [packages...]",
	Short: "Download packages without installing",
	Long:  `Downloads the specified packages to the cache without installing them.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, err := NewEngineWithConfig(nil)
		if err != nil {
			return fmt.Errorf("failed to initialize engine: %w", err)
		}

		if err := RunDownload(eng, cmd.OutOrStdout(), args); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Downloaded packages successfully.")
		return nil
	},
}

// RunDownload executes the download logic
func RunDownload(eng *core.Engine, w io.Writer, pkgs []string) error {
	fmt.Fprintln(w, "Downloading packages...")
	options := core.InstallOptions{
		DownloadOnly: true,
		// Standard install options apply
	}
	return eng.Install(pkgs, options)
}
