package main

import (
	"fmt"
	"os"

	"github.com/theshedman/shedman/cmd/shedman/commands"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/output"
	"github.com/theshedman/shedman/internal/signals"
)

// Version information - set at build time
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

// Global flags
var (
	YesFlag       bool
	NoconfirmFlag bool // Alias for YesFlag (pacman compat)
	QuietFlag     bool
	VerboseFlag   bool
	DebugFlag     bool
	DryRunFlag    bool
	ColorFlag     bool
	NoColorFlag   bool
	ConfigFile    string
var RootCmd = &cobra.Command{

	Use:   "shedman",
	Short: "A universal package manager for ShedOS and beyond",
	Long:  `A modern package manager designed for ShedOS with pluggable backend architecture for seamless package management across other Linux distributions.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize signal handling for cleanup
		signals.SetupSignalHandler()

		// Handle noconfirm as alias for yes
		if NoconfirmFlag {
			YesFlag = true
		}
		// Initialize color output
		output.InitColor(ColorFlag, NoColorFlag)
	},
}

// Execute runs the root command
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Setup colored help templates
	output.SetupColoredHelp(rootCmd)

	// Add subcommands here
	rootCmd.AddCommand(commands.VersionCmd)

	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&YesFlag, "yes", "y", false, "Skip all confirmations")
	rootCmd.PersistentFlags().BoolVar(&NoconfirmFlag, "noconfirm", false, "Alias for --yes (pacman compat)")
	rootCmd.PersistentFlags().BoolVarP(&QuietFlag, "quiet", "q", false, "Minimal output")
	rootCmd.PersistentFlags().BoolVarP(&VerboseFlag, "verbose", "v", false, "Detailed output")
	rootCmd.PersistentFlags().BoolVar(&DebugFlag, "debug", false, "Developer debug output")
	rootCmd.PersistentFlags().BoolVar(&DryRunFlag, "dry-run", false, "Preview without executing")
	rootCmd.PersistentFlags().BoolVar(&ColorFlag, "color", false, "Force colored output")
	rootCmd.PersistentFlags().BoolVar(&NoColorFlag, "no-color", false, "Disable colors")
	rootCmd.PersistentFlags().StringVarP(&ConfigFile, "config", "c", "", "Custom config file path")
}

func main() {
	Execute()
}
