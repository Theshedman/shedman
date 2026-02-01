package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/config"
)

// HoldCmd represents the hold command.
var HoldCmd = &cobra.Command{
	Use:   "hold <package>",
	Short: "Hold a package back from upgrades",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefault()
		if err != nil {
			return err
		}

		pkg := strings.TrimSpace(args[0])
		if pkg == "" {
			return fmt.Errorf("package name is required")
		}

		if contains(cfg.Packages.HoldPkg, pkg) {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s is already held.\n", pkg)
			return nil
		}

		cfg.Packages.HoldPkg = append(cfg.Packages.HoldPkg, pkg)
		if err := config.Save(config.DefaultConfigPath(), cfg); err != nil {
			return err
		}

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Held package: %s\n", pkg)
		return nil
	},
}

var holdListCmd = &cobra.Command{
	Use:   "list",
	Short: "List held packages",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefault()
		if err != nil {
			return err
		}
		if len(cfg.Packages.HoldPkg) == 0 {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No held packages.")
			return nil
		}
		for _, pkg := range cfg.Packages.HoldPkg {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), pkg)
		}
		return nil
	},
}

func init() {
	HoldCmd.AddCommand(holdListCmd)
}

func contains(list []string, val string) bool {
	for _, item := range list {
		if item == val {
			return true
		}
	}
	return false
}
