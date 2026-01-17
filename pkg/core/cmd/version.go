package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints the current version",
	Long:  "Prints the current version",
	Run: func(cmd *cobra.Command, args []string) {
		RunVersion(cmd.OutOrStdout())
	},
}

func RunVersion(w io.Writer) {
	_, _ = fmt.Fprintln(w, "shedman version", Version)
	_, _ = fmt.Fprintln(w, "Build Date:", BuildDate)
	_, _ = fmt.Fprintln(w, "Git Commit:", GitCommit)

}
