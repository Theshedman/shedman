package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

func TestRunSecurityCheck_FiltersSeverity(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	mock.AuditFunc = func() ([]string, error) {
		return []string{
			"openssl 1.1 (CVE-2023-1234) HIGH: tls issue",
			"bash 5.1 (CVE-2020-1111) medium",
		}, nil
	}

	var buf bytes.Buffer
	opts := SecurityOptions{Severity: "high"}
	if err := RunSecurityCheck(eng, &buf, opts); err != nil {
		t.Fatalf("RunSecurityCheck failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "openssl") {
		t.Errorf("expected high severity package in output, got: %s", out)
	}
	if strings.Contains(out, "bash") {
		t.Errorf("did not expect medium severity package in output, got: %s", out)
	}
}

func TestRunSecurityFix_UpgradesPackages(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	mock.AuditFunc = func() ([]string, error) {
		return []string{
			"openssl 1.1 (CVE-2023-1234) HIGH: tls issue",
			"openssl 1.1 (CVE-2023-9999) HIGH: other issue",
			"bash 5.1 (CVE-2020-1111) medium",
		}, nil
	}

	upgradeCalled := false
	mock.UpgradeFunc = func(pkgs []string, _ core.UpgradeOptions) error {
		upgradeCalled = true
		if len(pkgs) != 2 {
			t.Fatalf("expected 2 packages to upgrade, got %v", pkgs)
		}
		return nil
	}

	var buf bytes.Buffer
	if err := RunSecurityFix(eng, &buf, SecurityOptions{}, true); err != nil {
		t.Fatalf("RunSecurityFix failed: %v", err)
	}

	if !upgradeCalled {
		t.Error("expected upgrade to be called")
	}
}

func TestRunSecurityCheck_NoVulns(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	mock.AuditFunc = func() ([]string, error) {
		return []string{}, nil
	}

	var buf bytes.Buffer
	if err := RunSecurityCheck(eng, &buf, SecurityOptions{}); err != nil {
		t.Fatalf("RunSecurityCheck failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "No vulnerabilities found") {
		t.Errorf("Expected success message, got: %s", out)
	}
}
