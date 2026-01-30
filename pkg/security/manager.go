package security

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/theshedman/shedman/pkg/core"
)

// Scanner handles security scanning
type Scanner struct {
	core *core.Engine
}

// New creates a new security scanner
func New(c *core.Engine) *Scanner {
	return &Scanner{
		core: c,
	}
}

// Vulnerability represents a security vulnerability
type Vulnerability struct {
	CVE      string
	Package  string
	Severity string
}

var (
	cvePattern      = regexp.MustCompile(`CVE-\d{4}-\d{4,7}`)
	severityPattern = regexp.MustCompile(`(?i)\b(critical|high|medium|low)\b`)
)

// Check checks for vulnerabilities
func (s *Scanner) Check() ([]Vulnerability, error) {
	if s.core == nil {
		return nil, fmt.Errorf("security scanner requires a core engine")
	}

	issues, err := s.core.Audit()
	if err != nil {
		return nil, err
	}

	var vulns []Vulnerability
	for _, line := range issues {
		vulns = append(vulns, parseAuditLine(line)...)
	}

	return vulns, nil
}

func parseAuditLine(line string) []Vulnerability {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	fields := strings.Fields(line)
	pkg := ""
	if len(fields) > 0 {
		pkg = fields[0]
	}

	severity := ""
	if match := severityPattern.FindString(line); match != "" {
		severity = strings.ToLower(match)
	}

	cves := cvePattern.FindAllString(line, -1)
	if len(cves) == 0 {
		return []Vulnerability{{
			CVE:      "",
			Package:  pkg,
			Severity: severity,
		}}
	}

	seen := make(map[string]bool)
	var vulns []Vulnerability
	for _, cve := range cves {
		if seen[cve] {
			continue
		}
		seen[cve] = true
		vulns = append(vulns, Vulnerability{
			CVE:      cve,
			Package:  pkg,
			Severity: severity,
		})
	}

	return vulns
}
