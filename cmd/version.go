package cmd

import "github.com/spf13/cobra"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints the current version of the shedman package manager",
	Long:  "Prints the current version of the shedman packange manager",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println("shedman version", Version)
		cmd.Println("Build Date:", BuildDate)
		cmd.Println("Git Commit:", GitCommit)
	},
}
