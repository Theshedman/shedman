package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/output"
	"github.com/theshedman/shedman/pkg/core"
)

// VerifyCmd represents the verify command
var VerifyCmd = &cobra.Command{
	Use:   "verify [package]",
	Short: "Check package integrity",
	Long:  `Verify the presence and integrity of all files owned by a package. If no package is specified, verifies all installed packages.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		eng, err := NewEngineWithConfig(nil)
		if err != nil {
			output.Error("Failed to initialize engine: %v", err)
			return
		}

		fix, _ := cmd.Flags().GetBool("fix")
		if err := RunVerify(eng, args, fix); err != nil {
			output.Error("%v", err)
		}
	},
}

func init() {
	VerifyCmd.Flags().Bool("fix", false, "Reinstall corrupted packages")
}

// RunVerify executes the verify logic
func RunVerify(eng *core.Engine, args []string, fix bool) error {
	if len(args) == 0 {
		return verifyAll(eng, fix)
	}
	return verifyOne(eng, args[0], fix)
}

func verifyOne(eng *core.Engine, pkg string, fix bool) error {
	output.Info("Verifying %s...", pkg)
	issues, err := eng.VerifyPackage(pkg)
	if err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	if len(issues) == 0 {
		output.Success("Package '%s' is intact.", pkg)
		return nil
	}

	output.Warning("Found %d integrity issues for %s:", len(issues), pkg)
	for _, issue := range issues {
		fmt.Printf("  - %s\n", issue)
	}

	if fix {
		output.Info("Attempting to fix %s by reinstalling...", pkg)
		opts := core.InstallOptions{
			NoConfirm: true, // Fix implies auto-confirm for reinstallation
		}
		if err := eng.Install([]string{pkg}, opts); err != nil {
			return fmt.Errorf("failed to reinstall %s: %w", pkg, err)
		}
		output.Success("Reinstalled %s.", pkg)
	}

	return nil
}

func verifyAll(eng *core.Engine, fix bool) error {
	output.Info("Verifying all packages...")
	results, err := eng.VerifyAll()
	if err != nil {
		return fmt.Errorf("full verification failed: %w", err)
	}

	if len(results) == 0 {
		output.Success("All packages are intact.")
		return nil
	}

	output.Warning("Found issues in %d packages.", len(results))
	for pkg, issues := range results {
		fmt.Printf("%s:\n", pkg)
		for _, issue := range issues {
			fmt.Printf("  - %s\n", issue)
		}

		if fix {
			output.Info("Fixing %s...", pkg)
			if err := eng.Install([]string{pkg}, core.InstallOptions{NoConfirm: true}); err != nil {
				output.Error("Failed to fix %s: %v", pkg, err)
				// Continue with others
			} else {
				output.Success("Fixed %s.", pkg)
			}
		}
	}
	return nil
}
