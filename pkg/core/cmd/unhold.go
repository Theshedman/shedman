package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/config"
)

// UnholdCmd represents the unhold command.
var UnholdCmd = &cobra.Command{
	Use:   "unhold <package>",
	Short: "Remove a package from the hold list",
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

		var updated []string
		found := false
		for _, item := range cfg.Packages.HoldPkg {
			if item == pkg {
				found = true
				continue
			}
			updated = append(updated, item)
		}

		if !found {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s is not held.\n", pkg)
			return nil
		}

		cfg.Packages.HoldPkg = updated
		if err := config.Save(config.DefaultConfigPath(), cfg); err != nil {
			return err
		}

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Unheld package: %s\n", pkg)
		return nil
	},
}
