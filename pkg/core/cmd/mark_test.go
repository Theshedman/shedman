package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

func TestRunMark(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	markCalled := false
	var capturedPkg string
	var capturedReason core.InstallReason

	mock.SetInstallReasonFunc = func(pkg string, reason core.InstallReason) error {
		markCalled = true
		capturedPkg = pkg
		capturedReason = reason
		return nil
	}

	// Test --as-deps
	var buf bytes.Buffer
	if err := RunMark(eng, &buf, "vim", true, false); err != nil {
		t.Fatalf("RunMark failed: %v", err)
	}

	if !markCalled {
		t.Error("SetInstallReason was not called")
	}
	if capturedPkg != "vim" {
		t.Errorf("Expected pkg 'vim', got '%s'", capturedPkg)
	}
	if capturedReason != core.InstallReasonDependency {
		t.Errorf("Expected reason dependency, got %v", capturedReason)
	}
	if !strings.Contains(buf.String(), "Marking vim as dependency") {
		t.Errorf("Unexpected output: %s", buf.String())
	}

	// Reset
	markCalled = false
	buf.Reset()

	// Test --as-explicit
	if err := RunMark(eng, &buf, "nano", false, true); err != nil {
		t.Fatalf("RunMark failed: %v", err)
	}

	if !markCalled {
		t.Error("SetInstallReason was not called")
	}
	if capturedPkg != "nano" {
		t.Errorf("Expected pkg 'nano', got '%s'", capturedPkg)
	}
	if capturedReason != core.InstallReasonExplicit {
		t.Errorf("Expected reason explicit, got %v", capturedReason)
	}
	if !strings.Contains(buf.String(), "Marking nano as explicit") {
		t.Errorf("Unexpected output: %s", buf.String())
	}
}
