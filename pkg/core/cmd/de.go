package cmd

import (
	"github.com/spf13/cobra"
)

// DeCmd represents the de (desktop environment) command
var DeCmd = &cobra.Command{
	Use:   "de",
	Short: "Manage desktop environments",
	Long:  `List and switch between Desktop Environments (DEs) with automated snapshotting and configuration.`,
}

func init() {
	DeCmd.AddCommand(deListCmd)
	DeCmd.AddCommand(deSwitchCmd)
}
