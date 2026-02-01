package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/internal/output"
	"github.com/theshedman/shedman/internal/signals"
	configcmd "github.com/theshedman/shedman/pkg/config/cmd"
	commands "github.com/theshedman/shedman/pkg/core/cmd"
	"github.com/theshedman/shedman/pkg/logger"
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
	apiFlag       bool
	apiAddrFlag   string
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

		// Initialize structured logger
		logger.Init(debugFlag, verboseFlag)

		// Ensure config exists (auto-create if missing)
		if configFile == "" {

			_, _ = config.LoadDefault()
		} else {
			_, _ = config.Load(configFile)
		}

		if apiFlag {
			eng, err := commands.NewEngineWithConfig(nil)
			if err != nil {
				_, _ = fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			if err := commands.RunAPI(eng, apiAddrFlag); err != nil {
				_, _ = fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			os.Exit(0)
		}
	},
}

// Execute runs the root command
func Execute() {
	rootCmd.SetArgs(rewritePacmanArgs(os.Args[1:]))
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)

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
	rootCmd.AddCommand(commands.InfoCmd)
	rootCmd.AddCommand(commands.GroupCmd)
	rootCmd.AddCommand(commands.CleanCmd)
	rootCmd.AddCommand(commands.OrphansCmd)
	rootCmd.AddCommand(commands.OwnsCmd)
	rootCmd.AddCommand(commands.DoctorCmd)
	rootCmd.AddCommand(commands.HistoryCmd)
	rootCmd.AddCommand(commands.WhyCmd)
	rootCmd.AddCommand(commands.VerifyCmd)
	rootCmd.AddCommand(commands.BuildCmd)
	rootCmd.AddCommand(commands.KeyringCmd)
	rootCmd.AddCommand(commands.RepairCmd)
	rootCmd.AddCommand(commands.FilesCmd)
	rootCmd.AddCommand(commands.MarkCmd)
	rootCmd.AddCommand(commands.SizeCmd)
	rootCmd.AddCommand(commands.CheckCmd)
	rootCmd.AddCommand(commands.DiffCmd)
	rootCmd.AddCommand(commands.DownloadCmd)
	rootCmd.AddCommand(commands.ExportCmd)
	rootCmd.AddCommand(commands.ImportCmd)
	rootCmd.AddCommand(commands.LogCmd)
	rootCmd.AddCommand(commands.SecurityCmd)
	rootCmd.AddCommand(commands.BootCmd)
	rootCmd.AddCommand(commands.ThemeCmd)
	rootCmd.AddCommand(commands.SnapshotCmd)
	rootCmd.AddCommand(commands.DeCmd)
	rootCmd.AddCommand(commands.SvcCmd)
	rootCmd.AddCommand(commands.MirrorCmd)
	rootCmd.AddCommand(commands.NotifierCmd)
	rootCmd.AddCommand(commands.TUICmd)
	rootCmd.AddCommand(commands.CompletionCmd)
	rootCmd.AddCommand(commands.HoldCmd)
	rootCmd.AddCommand(commands.UnholdCmd)
	rootCmd.AddCommand(commands.MigrateCmd)
	rootCmd.AddCommand(commands.ApiCmd)
	rootCmd.AddCommand(configcmd.ConfigCmd)

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
	rootCmd.PersistentFlags().BoolVar(&apiFlag, "api", false, "Start API server mode")
	rootCmd.PersistentFlags().StringVar(&apiAddrFlag, "api-addr", "127.0.0.1:7337", "API listen address")
}

func main() {
	Execute()
}
