package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/pkg/boot"
)

// BootCmd represents the boot command group.
var BootCmd = &cobra.Command{
	Use:   "boot",
	Short: "Manage boot entries and kernels",
	Long:  "List installed kernels and manage bootloader default/oneshot entries.",
}

var bootListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed kernels",
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}
		mgr := boot.New(engine)
		kernels, err := mgr.List()
		if err != nil {
			return err
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
		_, _ = fmt.Fprintln(w, "NAME\tVERSION\tSTATUS")
		for _, k := range kernels {
			status := ""
			if k.Current {
				status = "current"
			}
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", k.Name, k.Version, status)
		}
		_ = w.Flush()
		return nil
	},
}

var bootSetDefaultCmd = &cobra.Command{
	Use:   "set-default <kernel>",
	Short: "Set the default kernel",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}
		mgr := boot.New(engine)
		if err := mgr.SetDefault(args[0]); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Default kernel set to %s.\n", args[0])
		return nil
	},
}

var bootSetOneshotCmd = &cobra.Command{
	Use:   "set-oneshot <kernel>",
	Short: "Set the next boot to use a kernel once",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}
		mgr := boot.New(engine)
		if err := mgr.SetOneshot(args[0]); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Next boot set to %s.\n", args[0])
		return nil
	},
}

func init() {
	BootCmd.AddCommand(bootListCmd)
	BootCmd.AddCommand(bootSetDefaultCmd)
	BootCmd.AddCommand(bootSetOneshotCmd)
}
