package security

import "github.com/theshedman/shedman/pkg/core"

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

// Check checks for vulnerabilities
func (s *Scanner) Check() ([]Vulnerability, error) {
	return nil, nil
}
