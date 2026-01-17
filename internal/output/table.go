package output

import (
	"fmt"
	"regexp"
	"strings"
)

// ansiRegex matches ANSI escape codes
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// visibleLen returns the visible length of a string (excluding ANSI codes)
func visibleLen(s string) int {
	return len(stripANSI(s))
}

// Table represents a formatted table for terminal output
type Table struct {
	headers  []string
	rows     [][]string
	colWidth []int
}

// NewTable creates a new table with headers
func NewTable(headers ...string) *Table {
	t := &Table{
		headers:  headers,
		rows:     make([][]string, 0),
		colWidth: make([]int, len(headers)),
	}
	// Initialize column widths from headers
	for i, h := range headers {
		t.colWidth[i] = visibleLen(h)
	}
	return t
}

// AddRow adds a row to the table
func (t *Table) AddRow(cells ...string) {
	// Pad or truncate to match header count
	row := make([]string, len(t.headers))
	for i := range row {
		if i < len(cells) {
			row[i] = cells[i]
		} else {
			row[i] = ""
		}
		// Update column width using visible length
		if visibleLen(row[i]) > t.colWidth[i] {
			t.colWidth[i] = visibleLen(row[i])
		}
	}
	t.rows = append(t.rows, row)
}

// Render returns the table as a formatted string
func (t *Table) Render() string {
	var sb strings.Builder

	// Header
	t.renderRow(&sb, t.headers, Bold+Cyan)

	// Separator
	separator := make([]string, len(t.headers))
	for i, w := range t.colWidth {
		separator[i] = strings.Repeat("─", w)
	}
	sb.WriteString("├" + strings.Join(separator, "┼") + "┤\n")

	// Rows
	for _, row := range t.rows {
		t.renderRow(&sb, row, "")
	}

	return sb.String()
}

// renderRow renders a single row
func (t *Table) renderRow(sb *strings.Builder, cells []string, color string) {
	parts := make([]string, len(cells))
	for i, cell := range cells {
		padded := fmt.Sprintf("%-*s", t.colWidth[i], cell)
		if color != "" && colorEnabled {
			parts[i] = color + padded + Reset
		} else {
			parts[i] = padded
		}
	}
	sb.WriteString("│" + strings.Join(parts, "│") + "│\n")
}

// Print outputs the table to stdout
func (t *Table) Print() {
	_, _ = fmt.Print(t.Render())

}

// InstallSummaryTable creates a summary table for package installation
type InstallSummaryTable struct {
	packages   []SummaryRow
	totalSize  int64
	totalCount int
}

// SummaryRow represents a single package in the summary
type SummaryRow struct {
	Name    string
	Version string
	Source  string
	Size    int64
	Status  string // "install", "upgrade", "reinstall"
}

// NewInstallSummaryTable creates a new install summary
func NewInstallSummaryTable() *InstallSummaryTable {
	return &InstallSummaryTable{
		packages: make([]SummaryRow, 0),
	}
}

// AddPackage adds a package to the summary
func (s *InstallSummaryTable) AddPackage(row SummaryRow) {
	s.packages = append(s.packages, row)
	s.totalSize += row.Size
	s.totalCount++
}

// Render returns the formatted summary
func (s *InstallSummaryTable) Render() string {
	if len(s.packages) == 0 {
		return "No packages to install.\n"
	}

	var sb strings.Builder

	// Header
	sb.WriteString(Colorize(Bold, "Packages to install:\n"))
	sb.WriteString("\n")

	// Create table
	table := NewTable("Name", "Version", "Source", "Size", "Status")

	for _, pkg := range s.packages {
		sizeStr := formatSize(pkg.Size)
		sourceBadge := SourceBadge(pkg.Source)
		table.AddRow(pkg.Name, pkg.Version, sourceBadge, sizeStr, pkg.Status)
	}

	sb.WriteString(table.Render())

	// Summary line
	sb.WriteString(fmt.Sprintf("\n%s: %d packages, %s total\n",
		Colorize(Bold, "Summary"),
		s.totalCount,
		formatSize(s.totalSize)))

	return sb.String()
}

// Print outputs the summary
func (s *InstallSummaryTable) Print() {
	_, _ = fmt.Print(s.Render())

}

// SourceBadge returns a colored source badge
func SourceBadge(source string) string {
	switch source {
	case "official":
		return Colorize(Green, "[official]")
	case "aur":
		return Colorize(Magenta, "[aur]")
	case "shedos":
		return Colorize(Cyan, "[shedos]")
	default:
		return "[" + source + "]"
	}
}

// formatSize formats bytes into human-readable size
func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
