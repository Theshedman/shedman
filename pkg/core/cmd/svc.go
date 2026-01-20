package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/pkg/svc"
)

// SvcCmd represents the svc command
var SvcCmd = &cobra.Command{
	Use:   "svc",
	Short: "Manage system services",
	Long: `Manage system services (enable, disable, start, stop, status).
This simplifies service management on ShedOS.`,
}

// svcListCmd represents the list command
var svcListCmd = &cobra.Command{
	Use:   "list",
	Short: "List services",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Filter arg?
		// for now list all
		m := svc.New()
		services, err := m.List()
		if err != nil {
			return err
		}

		fmt.Printf("%-30s %-10s %-10s\n", "NAME", "ACTIVE", "ENABLED")
		for _, s := range services {
			active := "inactive"
			if s.Active {
				active = "active"
			}
			enabled := "disabled"
			if s.Enabled {
				enabled = "enabled"
			}
			fmt.Printf("%-30s %-10s %-10s\n", s.Name, active, enabled)
		}
		return nil
	},
}

var svcEnableCmd = &cobra.Command{
	Use:   "enable [service]",
	Short: "Enable and start a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		m := svc.New()
		if err := m.Enable(name); err != nil {
			return err
		}
		fmt.Printf("Service %s enabled and started.\n", name)
		return nil
	},
}

var svcDisableCmd = &cobra.Command{
	Use:   "disable [service]",
	Short: "Disable and stop a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		m := svc.New()
		if err := m.Disable(name); err != nil {
			return err
		}
		fmt.Printf("Service %s disabled and stopped.\n", name)
		return nil
	},
}

var svcStartCmd = &cobra.Command{
	Use:   "start [service]",
	Short: "Start a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		m := svc.New()
		return m.Start(name)
	},
}

var svcStopCmd = &cobra.Command{
	Use:   "stop [service]",
	Short: "Stop a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		m := svc.New()
		return m.Stop(name)
	},
}

var svcStatusCmd = &cobra.Command{
	Use:   "status [service]",
	Short: "Show service status",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		m := svc.New()
		s, err := m.Status(name)
		if err != nil {
			return err
		}
		fmt.Printf("Service: %s\n", s.Name)
		fmt.Printf("Active: %v\n", s.Active)
		fmt.Printf("Enabled: %v\n", s.Enabled)
		return nil
	},
}

func init() {
	SvcCmd.AddCommand(svcListCmd)
	SvcCmd.AddCommand(svcEnableCmd)
	SvcCmd.AddCommand(svcDisableCmd)
	SvcCmd.AddCommand(svcStartCmd)
	SvcCmd.AddCommand(svcStopCmd)
	SvcCmd.AddCommand(svcStatusCmd)
}
