package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/output"
	"github.com/theshedman/shedman/pkg/core"
)

// GroupCmd represents the group command
var GroupCmd = &cobra.Command{
	Use:   "group",
	Short: "Manage package groups",
	Long:  `List, install, remove, or view information about package groups.`,
}

var groupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available package groups",
	Run: func(cmd *cobra.Command, args []string) {
		eng, err := NewEngineWithConfig(nil)
		if err != nil {
			output.Error("Failed to initialize engine: %v", err)
			return
		}

		if err := RunGroupList(eng, os.Stdout); err != nil {
			output.Error("%v", err)
		}
	},
}

var groupInfoCmd = &cobra.Command{
	Use:   "info [group]",
	Short: "Show packages within a group",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		eng, err := NewEngineWithConfig(nil)
		if err != nil {
			output.Error("Failed to initialize engine: %v", err)
			return
		}

		if err := RunGroupInfo(eng, os.Stdout, args[0]); err != nil {
			output.Error("%v", err)
		}
	},
}

var groupInstallCmd = &cobra.Command{
	Use:   "install [group]",
	Short: "Install all packages in a group",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		eng, err := NewEngineWithConfig(nil)
		if err != nil {
			output.Error("Failed to initialize engine: %v", err)
			return
		}

		if err := RunGroupInstall(eng, os.Stdout, args[0]); err != nil {
			output.Error("Installation failed: %v", err)
		}
	},
}

var groupRemoveCmd = &cobra.Command{
	Use:   "remove [group]",
	Short: "Remove all packages in a group",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		eng, err := NewEngineWithConfig(nil)
		if err != nil {
			output.Error("Failed to initialize engine: %v", err)
			return
		}

		if err := RunGroupRemove(eng, os.Stdout, args[0]); err != nil {
			output.Error("Removal failed: %v", err)
		}
	},
}

func init() {
	GroupCmd.AddCommand(groupListCmd)
	GroupCmd.AddCommand(groupInfoCmd)
	GroupCmd.AddCommand(groupInstallCmd)
	GroupCmd.AddCommand(groupRemoveCmd)
}

// RunGroupList executes group list logic
func RunGroupList(eng *core.Engine, w io.Writer) error {
	groups, err := eng.ListGroups()
	if err != nil {
		return fmt.Errorf("failed to list groups: %w", err)
	}

	fmt.Fprintln(w, "Available Groups:")
	for _, g := range groups {
		fmt.Fprintln(w, "  "+g)
	}
	return nil
}

// RunGroupInfo executes group info logic
func RunGroupInfo(eng *core.Engine, w io.Writer, groupName string) error {
	groupName = cleanGroupName(groupName)
	pkgs, err := eng.GetGroupPackages(groupName)
	if err != nil {
		return fmt.Errorf("failed to get group info: %w", err)
	}

	_, _ = fmt.Fprintf(w, "Group: %s\n", groupName)

	fmt.Fprintf(w, "Packages (%d):\n", len(pkgs))
	for _, p := range pkgs {
		fmt.Fprintln(w, "  "+p)
	}
	return nil
}

// RunGroupInstall executes group install logic
func RunGroupInstall(eng *core.Engine, w io.Writer, groupName string) error {
	groupName = cleanGroupName(groupName)
	fmt.Fprintf(w, "Resolving group %s...\n", groupName)

	pkgs, err := eng.GetGroupPackages(groupName)
	if err != nil {
		return fmt.Errorf("failed to resolve group: %w", err)
	}

	fmt.Fprintf(w, "Installing %d packages from group %s...\n", len(pkgs), groupName)
	return eng.Install(pkgs, core.InstallOptions{})
}

// RunGroupRemove executes group remove logic
func RunGroupRemove(eng *core.Engine, w io.Writer, groupName string) error {
	groupName = cleanGroupName(groupName)
	fmt.Fprintf(w, "Resolving group %s...\n", groupName)

	pkgs, err := eng.GetGroupPackages(groupName)
	if err != nil {
		return fmt.Errorf("failed to resolve group: %w", err)
	}

	fmt.Fprintf(w, "Removing %d packages from group %s...\n", len(pkgs), groupName)
	return eng.Remove(pkgs, core.RemoveOptions{})
}

func cleanGroupName(name string) string {
	if len(name) > 0 && name[0] == '@' {
		return name[1:]
	}
	return name
}
