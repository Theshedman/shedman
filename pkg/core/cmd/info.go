package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/internal/output"
	"github.com/theshedman/shedman/pkg/core"
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
			cfg, err := config.LoadDefault()
			if err != nil {
				output.Warning("Failed to load config, using defaults: %v", err)
				cfg = config.Default()
			}

			eng, err := NewEngineWithConfig(cfg)
			if err != nil {
				return fmt.Errorf("failed to initialize engine: %w", err)
			}

			return RunInfo(eng, cmd.OutOrStdout(), args[0], infoJSON)
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&infoJSON, "json", false, "Output in JSON format")

	return cmd
}

// RunInfo executes the info logic
func RunInfo(eng *core.Engine, w io.Writer, pkgName string, asJson bool) error {
	info, err := eng.Info(pkgName)
	if err != nil {
		return fmt.Errorf("failed to get info for %s: %w", pkgName, err)
	}

	if asJson {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(info)
	}

	printKV(w, "Name", info.Name)
	printKV(w, "Version", info.Version)
	printKV(w, "Description", info.Description)
	printKV(w, "Source", string(info.Source))
	if info.PackageType != "" {
		printKV(w, "Type", info.PackageType)
	}
	printKV(w, "URL", info.DownloadURL)
	if len(info.Depends) > 0 {
		printKV(w, "Depends On", fmt.Sprintf("%v", info.Depends))
	}
	if len(info.OptDepends) > 0 {
		printKV(w, "Optional Deps", fmt.Sprintf("%v", info.OptDepends))
	}
	if info.Size > 0 {
		printKV(w, "Download Size", fmt.Sprintf("%.2f MiB", float64(info.Size)/1024/1024))
	}
	if info.InstalledSize > 0 {
		printKV(w, "Installed Size", fmt.Sprintf("%.2f MiB", float64(info.InstalledSize)/1024/1024))
	}

	return nil
}

// printKV formats key-value output to writer
func printKV(w io.Writer, key string, value string) {
	if value == "" {
		return
	}
	// Align keys: assume max key length ~15
	fmt.Fprintf(w, "%-15s : %s\n", key, value)
}
