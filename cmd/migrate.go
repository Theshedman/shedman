package cmd

import (
"fmt"
"os"
"path/filepath"

"github.com/pelletier/go-toml/v2"
"github.com/spf13/cobra"
"github.com/theshedman/shedman/pkg/shedman/config"
"github.com/theshedman/shedman/pkg/shedman/migrate"
)

var (
migrateFrom   string
migrateDryRun bool
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Import configuration from pacman.conf",
	Long: `Import settings from /etc/pacman.conf into shedman config.

This imports:
  - Mirror configuration
  - IgnorePkg/IgnoreGroup settings
  - ParallelDownloads setting
  - SigLevel settings`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Parse pacman.conf
		pacmanConf, err := migrate.ParsePacmanConf(migrateFrom)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", migrateFrom, err)
		}

		// Load existing shedman config or create default
		cfg, err := config.Load(configFile)
		if err != nil {
			return fmt.Errorf("failed to load shedman config: %w", err)
		}

		// Apply pacman settings to shedman config
		if len(pacmanConf.Mirrors) > 0 {
			cfg.Mirrors.Arch = pacmanConf.Mirrors
		}
		if len(pacmanConf.IgnorePkg) > 0 {
			cfg.Packages.IgnorePkg = pacmanConf.IgnorePkg
		}
		if len(pacmanConf.IgnoreGroup) > 0 {
			cfg.Packages.IgnoreGroup = pacmanConf.IgnoreGroup
		}
		if len(pacmanConf.HoldPkg) > 0 {
			cfg.Packages.HoldPkg = pacmanConf.HoldPkg
		}
		if pacmanConf.ParallelDownloads > 0 {
			cfg.Network.ParallelDownloads = pacmanConf.ParallelDownloads
		}
		if pacmanConf.SigLevel != "" {
			cfg.Security.SigLevel = pacmanConf.SigLevel
		}

		// Dry-run: show what would be imported
		if migrateDryRun || dryRunFlag {
			cmd.Println("Would import the following settings:")
			cmd.Printf("  Mirrors: %v\n", pacmanConf.Mirrors)
			cmd.Printf("  IgnorePkg: %v\n", pacmanConf.IgnorePkg)
			cmd.Printf("  IgnoreGroup: %v\n", pacmanConf.IgnoreGroup)
			cmd.Printf("  HoldPkg: %v\n", pacmanConf.HoldPkg)
			cmd.Printf("  ParallelDownloads: %d\n", pacmanConf.ParallelDownloads)
			cmd.Printf("  SigLevel: %s\n", pacmanConf.SigLevel)
			return nil
		}

		// Write updated config
		configPath := configFile
		if configPath == "" {
			home, _ := os.UserHomeDir()
			configPath = filepath.Join(home, ".config", "shedman", "config.toml")
		}

		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		// Marshal and write
		data, err := toml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}

		if err := os.WriteFile(configPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}

		cmd.Printf("Successfully imported settings from %s\n", migrateFrom)
		cmd.Printf("Config saved to %s\n", configPath)

		return nil
	},
}

func init() {
	migrateCmd.Flags().StringVar(&migrateFrom, "from", "/etc/pacman.conf", "Path to pacman.conf file")
	migrateCmd.Flags().BoolVar(&migrateDryRun, "dry-run", false, "Show what would be imported without making changes")
	rootCmd.AddCommand(migrateCmd)
}
