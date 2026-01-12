package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/pkg/config"
	"github.com/theshedman/shedman/pkg/tui"
)

// ConfigCmd is the root command for configuration management
var ConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration files",
	Long:  `Scan, diff, and reset tracked configuration files managed by shedman.`,
}

func init() {
	ConfigCmd.AddCommand(newStatusCmd())
	ConfigCmd.AddCommand(newDiffCmd())
	ConfigCmd.AddCommand(newResetCmd())
	ConfigCmd.AddCommand(newApplyCmd())
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
			path, _ := filepath.Abs(args[0])
			eng := createEngine()

			// Get Original
			original, err := eng.GetOriginal(path)
			if err != nil {
				fmt.Printf("Error retrieving original content: %v\n", err)
				os.Exit(1)
			}

			// Get Current
			current, err := os.ReadFile(path)
			if err != nil {
				fmt.Printf("Error reading file: %v\n", err)
				os.Exit(1)
			}

			// Generate Diff
			// Differ expects strings
			diff, err := eng.Differ.GenerateDiff(path, string(current), "package-default", string(original))
			if err != nil {
				fmt.Printf("Error generating diff: %v\n", err)
				os.Exit(1)
			}

			if diff == "" {
				fmt.Println("No differences found.")
			} else {
				fmt.Println(diff)
			}
		},
	}
}

func newResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset [path]",
		Short: "Reset a configuration file to the tracked package state",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			path, _ := filepath.Abs(args[0])
			eng := createEngine()

			// Get Original
			original, err := eng.GetOriginal(path)
			if err != nil {
				fmt.Printf("Error retrieving original content: %v\n", err)
				os.Exit(1)
			}

			// Backup
			fmt.Println("Backing up current configuration...")
			if _, err := eng.BackupMgr.Backup(path); err != nil {
				fmt.Printf("Backup failed: %v\n", err)
				os.Exit(1)
			}

			// Overwrite
			if err := os.WriteFile(path, original, 0644); err != nil {
				fmt.Printf("Failed to write file: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Successfully reset %s to package default.\n", path)
		},
	}
}

func newApplyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "apply [path]",
		Short: "Interactively apply package version of a config file (3-way merge)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			path, _ := filepath.Abs(args[0])
			eng := createEngine()

			// 1. Get Owner
			owner, err := eng.GetFileOwner(path)
			if err != nil {
				fmt.Printf("Error identifying owner: %v\n", err)
				os.Exit(1)
			}

			// 2. Get Original Content
			original, err := eng.GetOriginal(path)
			if err != nil {
				fmt.Printf("Error retrieving original content: %v\n", err)
				os.Exit(1)
			}

			// 3. Write to temp source file
			tmpFile, err := os.CreateTemp("", "shedman-apply-*.conf")
			if err != nil {
				fmt.Printf("Temp file creation failed: %v\n", err)
				os.Exit(1)
			}
			defer os.Remove(tmpFile.Name()) // Cleanup

			if _, err := tmpFile.Write(original); err != nil {
				fmt.Printf("Failed to write to temp file: %v\n", err)
				os.Exit(1)
			}
			tmpFile.Close()

			// 4. Apply
			fmt.Printf("Applying configuration for %s (Owner: %s)...\n", path, owner)
			if err := eng.Apply(owner, tmpFile.Name(), path); err != nil {
				fmt.Printf("Apply failed: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("Configuration applied successfully.")
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
	provider := config.NewPacmanSourceProvider(nil)

	return config.NewConfigEngine(stateMgr, backupMgr, differ, resolver, provider)
}
