package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	pkgconfig "github.com/theshedman/shedman/pkg/config"
)

// ConfigCmd is the root for config management commands
var ConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration files",
	Long:  `Scan, diff, and reset tracked configuration files managed by shedman.`,
}

var configStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of tracked configuration files",
	Run: func(cmd *cobra.Command, args []string) {
		// Use default state path
		home, _ := os.UserHomeDir()
		statePath := filepath.Join(home, ".local", "state", "shedman", "configs.json")

		stateMgr := pkgconfig.NewJSONStateManager(statePath)
		if err := stateMgr.Load(); err != nil {
			fmt.Printf("Failed to load state: %v\n", err)
			return
		}

		states := stateMgr.List()

		if len(states) == 0 {
			fmt.Println("No configuration files are currently tracked.")
			return
		}

		fmt.Printf("%-50s %-20s %s\n", "PATH", "VERSION", "LAST MODIFIED")
		fmt.Printf("%-50s %-20s %s\n", "----", "-------", "-------------")
		for _, s := range states {
			fmt.Printf("%-50s %-20s %s\n", s.Path, s.Version, s.LastModified.Format("2006-01-02 15:04:05"))
		}
	},
}

var configDiffCmd = &cobra.Command{
	Use:   "diff [path]",
	Short: "Show differences between user file and package default",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Limitation: Requires source content.
		fmt.Println("Error: 'config diff' requires the original package source which is not currently cached.")
		fmt.Println("This command is primarily used during 'install/update' interactions.")
		os.Exit(1)
	},
}

var configResetCmd = &cobra.Command{
	Use:   "reset [path]",
	Short: "Reset a configuration file to the tracked package state",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Error: 'config reset' requires the original package source which is not currently cached.")
		os.Exit(1)
	},
}

func init() {
	ConfigCmd.AddCommand(configStatusCmd)
	ConfigCmd.AddCommand(configDiffCmd)
	ConfigCmd.AddCommand(configResetCmd)
}
