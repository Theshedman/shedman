package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/internal/output"
)

var (
	infoJSON bool
)

var InfoCmd = NewInfoCmd()

func NewInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info [package]",
		Short: "Display package information",
		Long:  `Display detailed information about a package from configured repositories or installed packages.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInfo(args[0])
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&infoJSON, "json", false, "Output in JSON format")

	return cmd
}

func runInfo(pkgName string) error {
	// Load configuration
	cfg, err := config.LoadDefault()
	if err != nil {
		output.Warning("Failed to load config, using defaults: %v", err)
		cfg = config.Default()
	}

	// Initialize Engine
	eng, err := NewEngineWithConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize engine: %w", err)
	}

	// Query Engine for Info
	info, err := eng.Info(pkgName)
	if err != nil {
		return fmt.Errorf("failed to get info for %s: %w", pkgName, err)
	}

	if infoJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(info)
	}

	// Render Text Output
	output.PrintInfoKV("Name", info.Name)
	output.PrintInfoKV("Version", info.Version)
	output.PrintInfoKV("Description", info.Description)
	output.PrintInfoKV("Source", string(info.Source))
	if info.PackageType != "" {
		output.PrintInfoKV("Type", info.PackageType)
	}
	output.PrintInfoKV("URL", info.DownloadURL)
	if len(info.Depends) > 0 {
		output.PrintInfoKV("Depends On", fmt.Sprintf("%v", info.Depends))
	}
	if len(info.OptDepends) > 0 {
		output.PrintInfoKV("Optional Deps", fmt.Sprintf("%v", info.OptDepends))
	}
	if info.Size > 0 {
		output.PrintInfoKV("Download Size", fmt.Sprintf("%.2f MiB", float64(info.Size)/1024/1024))
	}
	if info.InstalledSize > 0 {
		output.PrintInfoKV("Installed Size", fmt.Sprintf("%.2f MiB", float64(info.InstalledSize)/1024/1024))
	}

	return nil
}

func init() {
}
