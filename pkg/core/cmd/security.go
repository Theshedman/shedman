package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/security"
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
		opts := SecurityOptions{
			JSON:     securityJSON,
			Severity: securitySeverity,
		}
		return RunSecurityCheck(eng, cmd.OutOrStdout(), opts)
	},
}

var (
	securityJSON     bool
	securitySeverity string
	securityYes      bool
)

type SecurityOptions struct {
	JSON     bool
	Severity string
}

type SecurityReport struct {
	Total           int                      `json:"total"`
	Packages        int                      `json:"packages"`
	BySeverity      map[string]int           `json:"by_severity"`
	Vulnerabilities []security.Vulnerability `json:"vulnerabilities"`
}

func init() {
	SecurityCmd.AddCommand(newSecurityCheckCmd())
	SecurityCmd.AddCommand(newSecurityListCmd())
	SecurityCmd.AddCommand(newSecurityFixCmd())
	SecurityCmd.AddCommand(newSecurityReportCmd())

	SecurityCmd.PersistentFlags().BoolVar(&securityJSON, "json", false, "Output as JSON")
	SecurityCmd.PersistentFlags().StringVar(&securitySeverity, "severity", "", "Filter by severity (low|medium|high|critical)")
}

func newSecurityCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Check installed packages for vulnerabilities",
		RunE: func(cmd *cobra.Command, args []string) error {
			eng, err := NewEngineWithConfig(nil)
			if err != nil {
				return err
			}
			opts := SecurityOptions{JSON: securityJSON, Severity: securitySeverity}
			return RunSecurityCheck(eng, cmd.OutOrStdout(), opts)
		},
	}
}

func newSecurityListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List known vulnerabilities affecting installed packages",
		RunE: func(cmd *cobra.Command, args []string) error {
			eng, err := NewEngineWithConfig(nil)
			if err != nil {
				return err
			}
			opts := SecurityOptions{JSON: securityJSON, Severity: securitySeverity}
			return RunSecurityCheck(eng, cmd.OutOrStdout(), opts)
		},
	}
}

func newSecurityFixCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fix",
		Short: "Update vulnerable packages",
		RunE: func(cmd *cobra.Command, args []string) error {
			eng, err := NewEngineWithConfig(nil)
			if err != nil {
				return err
			}
			opts := SecurityOptions{JSON: securityJSON, Severity: securitySeverity}
			return RunSecurityFix(eng, cmd.OutOrStdout(), opts, securityYes)
		},
	}
	cmd.Flags().BoolVarP(&securityYes, "yes", "y", false, "Skip confirmation for upgrades")
	return cmd
}

func newSecurityReportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "report",
		Short: "Generate a security report",
		RunE: func(cmd *cobra.Command, args []string) error {
			eng, err := NewEngineWithConfig(nil)
			if err != nil {
				return err
			}
			opts := SecurityOptions{JSON: securityJSON, Severity: securitySeverity}
			return RunSecurityReport(eng, cmd.OutOrStdout(), opts)
		},
	}
}

// RunSecurityCheck performs a security audit.
func RunSecurityCheck(eng *core.Engine, w io.Writer, opts SecurityOptions) error {
	vulns, err := scanVulnerabilities(eng, opts)
	if err != nil {
		return err
	}

	if len(vulns) == 0 {
		_, _ = fmt.Fprintln(w, "No vulnerabilities found. System is secure.")
		return nil
	}

	if opts.JSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(vulns)
	}

	_, _ = fmt.Fprintf(w, "Found %d vulnerabilities:\n", len(vulns))
	for _, v := range vulns {
		line := v.Package
		if v.CVE != "" {
			line += " " + v.CVE
		}
		if v.Severity != "" {
			line += " (" + v.Severity + ")"
		}
		_, _ = fmt.Fprintln(w, line)
	}
	return nil
}

// RunSecurityFix upgrades vulnerable packages.
func RunSecurityFix(eng *core.Engine, w io.Writer, opts SecurityOptions, noConfirm bool) error {
	vulns, err := scanVulnerabilities(eng, opts)
	if err != nil {
		return err
	}

	if len(vulns) == 0 {
		_, _ = fmt.Fprintln(w, "No vulnerable packages found.")
		return nil
	}

	pkgs := uniquePackages(vulns)
	if err := eng.Sync(); err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	if err := eng.Upgrade(pkgs, core.UpgradeOptions{NoConfirm: noConfirm}); err != nil {
		return fmt.Errorf("upgrade failed: %w", err)
	}

	_, _ = fmt.Fprintf(w, "Upgraded %d package(s).\n", len(pkgs))
	return nil
}

// RunSecurityReport outputs a report summary.
func RunSecurityReport(eng *core.Engine, w io.Writer, opts SecurityOptions) error {
	vulns, err := scanVulnerabilities(eng, opts)
	if err != nil {
		return err
	}

	report := buildReport(vulns)
	if opts.JSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}

	_, _ = fmt.Fprintf(w, "Total vulnerabilities: %d\n", report.Total)
	_, _ = fmt.Fprintf(w, "Packages affected: %d\n", report.Packages)

	if len(report.BySeverity) > 0 {
		keys := make([]string, 0, len(report.BySeverity))
		for k := range report.BySeverity {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			_, _ = fmt.Fprintf(w, "  %s: %d\n", key, report.BySeverity[key])
		}
	}

	return nil
}

func scanVulnerabilities(eng *core.Engine, opts SecurityOptions) ([]security.Vulnerability, error) {
	scanner := security.New(eng)
	vulns, err := scanner.Check()
	if err != nil {
		return nil, fmt.Errorf("audit failed: %w", err)
	}
	return filterVulnerabilities(vulns, opts.Severity), nil
}

func filterVulnerabilities(vulns []security.Vulnerability, severity string) []security.Vulnerability {
	sev := strings.ToLower(strings.TrimSpace(severity))
	if sev == "" {
		return vulns
	}

	filtered := make([]security.Vulnerability, 0, len(vulns))
	for _, v := range vulns {
		if strings.ToLower(v.Severity) == sev {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

func uniquePackages(vulns []security.Vulnerability) []string {
	seen := make(map[string]bool)
	var pkgs []string
	for _, v := range vulns {
		if v.Package == "" || seen[v.Package] {
			continue
		}
		seen[v.Package] = true
		pkgs = append(pkgs, v.Package)
	}
	sort.Strings(pkgs)
	return pkgs
}

func buildReport(vulns []security.Vulnerability) SecurityReport {
	report := SecurityReport{
		Total:           len(vulns),
		BySeverity:      make(map[string]int),
		Vulnerabilities: vulns,
	}

	seenPkgs := make(map[string]bool)
	for _, v := range vulns {
		if v.Package != "" {
			seenPkgs[v.Package] = true
		}
		if v.Severity != "" {
			report.BySeverity[v.Severity]++
		}
	}
	report.Packages = len(seenPkgs)
	return report
}
