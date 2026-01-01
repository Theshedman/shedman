package cmd

import (
"fmt"
"os"
"path/filepath"
"strings"

"github.com/pelletier/go-toml/v2"
"github.com/spf13/cobra"
"github.com/theshedman/shedman/pkg/shedman/config"
"github.com/theshedman/shedman/pkg/shedman/migrate"
)

var (
migrateFromPacman string
migrateFromApt    string
migrateFromDnf    string
migrateDryRun     bool
migrateAuto       bool
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Import configuration from another package manager",
	Long: `Import settings from another package manager's configuration.

Examples:
  shedman migrate --pacman              # Import from /etc/pacman.conf
  shedman migrate --pacman /path/to/pacman.conf
  shedman migrate --apt                 # Import from apt sources (Debian/Ubuntu)
  shedman migrate --dnf                 # Import from dnf.conf (Fedora)
  shedman migrate --auto                # Auto-detect based on distro

This imports:
  - Mirror configuration
  - IgnorePkg/IgnoreGroup settings
  - ParallelDownloads setting
  - SigLevel settings`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Determine which package manager to migrate from
		var source string
		var sourcePath string

		if migrateAuto {
			detected, path := detectDistro()
			if detected == "" {
				return fmt.Errorf("could not auto-detect package manager. Please specify --pacman, --apt, or --dnf")
			}
			source = detected
			sourcePath = path
			cmd.Printf("Auto-detected: %s (%s)\n", source, sourcePath)
		} else if migrateFromPacman != "" {
			source = "pacman"
			sourcePath = migrateFromPacman
			if sourcePath == "true" || sourcePath == "" {
				sourcePath = "/etc/pacman.conf"
			}
		} else if migrateFromApt != "" {
			source = "apt"
			sourcePath = migrateFromApt
			return fmt.Errorf("apt migration not yet implemented")
		} else if migrateFromDnf != "" {
			source = "dnf"
			sourcePath = migrateFromDnf
			return fmt.Errorf("dnf migration not yet implemented")
		} else {
			return fmt.Errorf("please specify a source: --pacman, --apt, --dnf, or --auto")
		}

		// Currently only pacman is implemented
		if source != "pacman" {
			return fmt.Errorf("%s migration not yet implemented", source)
		}

		// Parse pacman.conf
		pacmanConf, err := migrate.ParsePacmanConf(sourcePath)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", sourcePath, err)
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
			cmd.Printf("Would import from %s (%s):\n", source, sourcePath)
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

		cmd.Printf("Successfully imported settings from %s (%s)\n", source, sourcePath)
		cmd.Printf("Config saved to %s\n", configPath)

		return nil
	},
}

// detectDistro attempts to detect the Linux distribution and return appropriate package manager
func detectDistro() (string, string) {
	// Check for /etc/os-release
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "", ""
	}

	content := strings.ToLower(string(data))

	// Arch-based
	if strings.Contains(content, "arch") || strings.Contains(content, "manjaro") ||
		strings.Contains(content, "endeavour") || strings.Contains(content, "shedos") {
		if _, err := os.Stat("/etc/pacman.conf"); err == nil {
			return "pacman", "/etc/pacman.conf"
		}
	}

	// Debian/Ubuntu-based
	if strings.Contains(content, "debian") || strings.Contains(content, "ubuntu") ||
		strings.Contains(content, "mint") || strings.Contains(content, "pop") {
		if _, err := os.Stat("/etc/apt/sources.list"); err == nil {
			return "apt", "/etc/apt/sources.list"
		}
	}

	// Fedora/RHEL-based
	if strings.Contains(content, "fedora") || strings.Contains(content, "rhel") ||
		strings.Contains(content, "centos") || strings.Contains(content, "rocky") {
		if _, err := os.Stat("/etc/dnf/dnf.conf"); err == nil {
			return "dnf", "/etc/dnf/dnf.conf"
		}
	}

	return "", ""
}

func init() {
	migrateCmd.Flags().StringVar(&migrateFromPacman, "pacman", "", "Import from pacman.conf (default: /etc/pacman.conf)")
	migrateCmd.Flags().StringVar(&migrateFromApt, "apt", "", "Import from apt sources.list")
	migrateCmd.Flags().StringVar(&migrateFromDnf, "dnf", "", "Import from dnf.conf")
	migrateCmd.Flags().BoolVar(&migrateAuto, "auto", false, "Auto-detect package manager based on distro")
	migrateCmd.Flags().BoolVar(&migrateDryRun, "dry-run", false, "Show what would be imported without making changes")
	rootCmd.AddCommand(migrateCmd)
}
