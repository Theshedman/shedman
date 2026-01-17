package cmd

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/pkg/core"
)

var DiffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show details of pending updates",
	Long: `Display detailed information about pending system updates.
this includes version changes, size differences, and potential security vulnerabilities (CVEs).

Requires updated sync databases (run 'shedman update --refresh' first if needed).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}
		return RunDiff(eng, cmd.OutOrStdout())
	},
}

// RunDiff compares package lists
func RunDiff(eng *core.Engine, w io.Writer) error {
	diffs, err := eng.Diff()
	if err != nil {
		return fmt.Errorf("failed to get diff: %w", err)
	}

	if len(diffs) == 0 {
		_, _ = fmt.Fprintln(w, "No updates found. System is up to date.")

		return nil
	}

	_, _ = fmt.Fprintf(w, "Found %d pending updates:\n\n", len(diffs))

	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	_, _ = fmt.Fprintln(tw, "PACKAGE\tVERSION\tSIZE\tDELTA\tISSUES")

	for _, d := range diffs {
		versionDiff := fmt.Sprintf("%s -> %s", d.OldVersion, d.NewVersion)
		downloadSize := formatSize(d.DownloadSize)

		deltaSign := "+"
		if d.SizeDelta < 0 {
			deltaSign = "" // negative number includes sign
		}
		sizeDelta := fmt.Sprintf("%s%s", deltaSign, formatSize(d.SizeDelta))
		if d.SizeDelta == 0 {
			sizeDelta = "-"
		}

		issues := ""
		if len(d.CVEs) > 0 {
			issues = fmt.Sprintf("%d CVEs", len(d.CVEs))
		}
		if d.Pacnew {
			if issues != "" {
				issues += ", "
			}
			issues += "PACNEW"
		}
		if issues == "" {
			issues = "-"
		}

		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", d.Name, versionDiff, downloadSize, sizeDelta, issues)

	}
	_ = tw.Flush()

	// Detail CVEs if any
	hasCVEs := false
	for _, d := range diffs {
		if len(d.CVEs) > 0 {
			hasCVEs = true
			break
		}
	}

	if hasCVEs {
		_, _ = fmt.Fprintln(w, "\nSecurity Warnings:")

		for _, d := range diffs {
			if len(d.CVEs) > 0 {
				_, _ = fmt.Fprintf(w, "  %s: %v\n", d.Name, d.CVEs)

			}
		}
	}

	return nil
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
