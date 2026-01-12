package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/pkg/config"
	"github.com/theshedman/shedman/pkg/tui"
)

// NewCommand creates the config command hierarchy
func NewCommand() *cobra.Command {
	usage := "config"
	short := "Manage configuration files"
	long := `Scan, diff, and reset tracked configuration files managed by shedman.`

	cmd := &cobra.Command{
		Use:   usage,
		Short: short,
		Long:  long,
	}

	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newDiffCmd())
	cmd.AddCommand(newResetCmd())

	return cmd
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show status of tracked configuration files",
		Run: func(cmd *cobra.Command, args []string) {
			// Use default state path
			home, _ := os.UserHomeDir()
			statePath := filepath.Join(home, ".local", "state", "shedman", "configs.json")

			stateMgr := config.NewJSONStateManager(statePath)
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
}

func newDiffCmd() *cobra.Command {
	return &cobra.Command{
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
}

func newResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset [path]",
		Short: "Reset a configuration file to the tracked package state",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Error: 'config reset' requires the original package source which is not currently cached.")
			os.Exit(1)
		},
	}
}

// createEngine is a helper to instantiate the full engine for CLI usage
// This replaces the external factory for self-containment
func createEngine() *config.ConfigEngine {
	home, _ := os.UserHomeDir()
	statePath := filepath.Join(home, ".local", "state", "shedman", "configs.json")

	stateMgr := config.NewJSONStateManager(statePath)
	_ = stateMgr.Load()

	backupMgr := config.NewFileBackupManager()
	differ := config.NewDiffer()
	resolver := tui.NewConflictResolver()

	return config.NewConfigEngine(stateMgr, backupMgr, differ, resolver)
}
