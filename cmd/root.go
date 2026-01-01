package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version information - set at build time
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

// Global flags
var (
	yesFlag     bool
	quietFlag   bool
	verboseFlag bool
	debugFlag   bool
	dryRunFlag  bool
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
	rootCmd.PersistentFlags().BoolVarP(&quietFlag, "quiet", "q", false, "Minimal output")
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "Detailed output")
	rootCmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, "Developer debug output")
	rootCmd.PersistentFlags().BoolVar(&dryRunFlag, "dry-run", false, "Preview without executing")
}
