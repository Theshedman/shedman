package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/output"
)

// WhyDeps holds dependencies for the why command
type WhyDeps struct {
	LookPath func(file string) (string, error)
	RunCmd   func(name string, args ...string) error
}

var defaultWhyDeps = WhyDeps{
	LookPath: exec.LookPath,
	RunCmd: func(name string, args ...string) error {
		c := exec.Command(name, args...)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

var whyTree bool

// WhyCmd represents the why command
var WhyCmd = &cobra.Command{
	Use:   "why [package]",
	Short: "Show why a package is installed (dependency chain)",
	Long:  `Uses pactree to show the reverse dependency chain for a package. Use --tree for forward dependency tree.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := RunWhy(defaultWhyDeps, cmd.OutOrStdout(), args[0], whyTree); err != nil {
			output.Error("%v", err)
		}
	},
}

func init() {
	WhyCmd.Flags().BoolVar(&whyTree, "tree", false, "Show forward dependency tree instead of reverse")
}

// RunWhy executes the why logic
func RunWhy(deps WhyDeps, w io.Writer, pkg string, tree bool) error {
	// Check for pactree
	if _, err := deps.LookPath("pactree"); err != nil {
		return fmt.Errorf("pactree not found. Please install 'pacman-contrib'")
	}

	args := []string{"-u"}
	if !tree {
		fmt.Fprintf(w, "Reverse dependency chain for %s:\n", pkg)
		args = append([]string{"-r"}, args...)
	} else {
		fmt.Fprintf(w, "Dependency tree for %s:\n", pkg)
	}
	args = append(args, pkg)

	// pactree -r -u (reverse, unique) or pactree -u (forward)
	if err := deps.RunCmd("pactree", args...); err != nil {
		return fmt.Errorf("pactree failed: %w", err)
	}
	return nil
}
