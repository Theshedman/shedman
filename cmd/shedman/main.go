package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/cmd/shedman/commands"
	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/internal/output"
	"github.com/theshedman/shedman/internal/signals"
)

// Global flags
var (
	yesFlag       bool
	noconfirmFlag bool
	quietFlag     bool
	verboseFlag   bool
	debugFlag     bool
	dryRunFlag    bool
	colorFlag     bool
	noColorFlag   bool
	configFile    string
)

var rootCmd = &cobra.Command{
	Use:   "shedman",
	Short: "A package manager for ShedOS",
	Long:  `A modern package manager designed for ShedOS and other Arch-based distributions`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize signal handling for cleanup
		signals.SetupSignalHandler()

		// Handle noconfirm as alias for yes
		if noconfirmFlag {
			yesFlag = true
		}
		// Initialize color output
		output.InitColor(colorFlag, noColorFlag)

		// Ensure config exists (auto-create if missing)
		if configFile == "" {
			_, _ = config.LoadDefault()
		} else {
			_, _ = config.Load(configFile)
		}
	},
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Setup colored help templates
	output.SetupColoredHelp(rootCmd)

	// Add subcommands here
	rootCmd.AddCommand(commands.VersionCmd)
	rootCmd.AddCommand(commands.InstallCmd)
	rootCmd.AddCommand(commands.RemoveCmd)
	rootCmd.AddCommand(commands.SearchCmd)
	rootCmd.AddCommand(commands.SyncCmd)
	rootCmd.AddCommand(commands.UpdateCmd)
	rootCmd.AddCommand(commands.UpdateCmd)
	rootCmd.AddCommand(commands.InfoCmd)
	rootCmd.AddCommand(commands.ConfigCmd)

	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&yesFlag, "yes", "y", false, "Skip all confirmations")
	rootCmd.PersistentFlags().BoolVar(&noconfirmFlag, "noconfirm", false, "Alias for --yes (pacman compat)")
	rootCmd.PersistentFlags().BoolVarP(&quietFlag, "quiet", "q", false, "Minimal output")
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "Detailed output")
	rootCmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, "Developer debug output")
	rootCmd.PersistentFlags().BoolVar(&dryRunFlag, "dry-run", false, "Preview without executing")
	rootCmd.PersistentFlags().BoolVar(&colorFlag, "color", false, "Force colored output")
	rootCmd.PersistentFlags().BoolVar(&noColorFlag, "no-color", false, "Disable colors")
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Custom config file path")
}

func main() {
	Execute()
}
