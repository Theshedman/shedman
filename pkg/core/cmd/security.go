package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/pkg/core"
)

var SecurityCmd = &cobra.Command{
	Use:   "security",
	Short: "Audit installed packages for security vulnerabilities",
	Long: `Audit installed packages for known CVEs using backend capabilities (e.g. arch-audit).
This requires the underlying backend to support security auditing.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, err := NewEngineWithConfig(nil) // Use factory default
		if err != nil {
			return err
		}
		return RunSecurity(eng, cmd.OutOrStdout())
	},
}

// RunSecurity executes the security audit logic.
func RunSecurity(eng *core.Engine, w io.Writer) error {
	issues, err := eng.Audit()
	if err != nil {
		return fmt.Errorf("audit failed: %w", err)
	}

	if len(issues) == 0 {
		fmt.Fprintln(w, "No vulnerabilities found. System is secure.")
		return nil
	}

	fmt.Fprintf(w, "Found %d vulnerabilities:\n", len(issues))
	for _, issue := range issues {
		fmt.Fprintln(w, issue)
	}

	return nil
}
