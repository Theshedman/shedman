package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

func TestRunSecurity(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	mock.AuditFunc = func() ([]string, error) {
		return []string{"vuln1", "vuln2"}, nil
	}

	var buf bytes.Buffer
	err := RunSecurity(eng, &buf)
	if err != nil {
		t.Fatalf("RunSecurity failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "vuln1") || !strings.Contains(out, "vuln2") {
		t.Errorf("Expected vulnerabilities in output, got: %s", out)
	}
}

func TestRunSecurity_NoVulns(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	mock.AuditFunc = func() ([]string, error) {
		return []string{}, nil
	}

	var buf bytes.Buffer
	err := RunSecurity(eng, &buf)
	if err != nil {
		t.Fatalf("RunSecurity failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "No vulnerabilities found") {
		t.Errorf("Expected success message, got: %s", out)
	}
}
