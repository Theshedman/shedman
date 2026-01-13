package cmd

import (
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

	// Test --asdeps
	err := eng.SetInstallReason("vim", core.InstallReasonDependency)
	if err != nil {
		t.Errorf("SetInstallReason error = %v", err)
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

	// Reset
	markCalled = false

	// Test --asexplicit
	err = eng.SetInstallReason("nano", core.InstallReasonExplicit)
	if err != nil {
		t.Errorf("SetInstallReason error = %v", err)
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
}
