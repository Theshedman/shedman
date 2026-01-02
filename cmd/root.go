package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/pkg/shedman/output"
)

// Version information - set at build time
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

// Global flags
var (
	yesFlag       bool
	noconfirmFlag bool // Alias for yesFlag (pacman compat)
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
	Short: "A universal package manager for ShedOS and beyond",
	Long: `shedman is a next-generation package manager designed for ShedOS (Arch-based).

Features:
  • 100% pacman compatible — drop-in replacement for all pacman commands
  • AUR integration — seamless access to the Arch User Repository
  • Universal .shed packages — install packages from any Linux distro
  • Snapshots — backup and restore your entire system state
  • Cloud sync — never lose your packages, configs, or themes`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Handle noconfirm as alias for yes
		if noconfirmFlag {
			yesFlag = true
		}
		// Initialize color output
		output.InitColor(colorFlag, noColorFlag)
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
	// Add subcommands here
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(syncCmd)

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
