package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

func TestRunRemove(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	// Mock IsInstalled for initial check
	mock.IsInstalledFunc = func(name string) bool {
		return name == "test-pkg"
	}

	// Mock Remove to track calls
	removeCalled := false
	mock.RemoveFunc = func(pkgs []string, opts core.RemoveOptions) error {
		removeCalled = true
		if len(pkgs) != 1 || pkgs[0] != "test-pkg" {
			t.Errorf("Expected removal of [test-pkg], got %v", pkgs)
		}
		if !opts.NoConfirm {
			t.Error("Expected NoConfirm=true (passed via test options)")
		}
		return nil
	}

	var buf bytes.Buffer
	pkgs := []string{"test-pkg"}
	// We pass options directly to RunRemove to avoid CLI flag parsing constraints in unit tests
	opts := core.RemoveOptions{
		NoConfirm: true,
	}

	// This function signature doesn't exist yet, so this test ensures TDD flow
	// RunRemove(eng, writer, pkgs, opts)
	if err := RunRemove(eng, &buf, pkgs, opts); err != nil {
		t.Fatalf("RunRemove failed: %v", err)
	}

	if !removeCalled {
		t.Error("Backend.Remove was not called")
	}

	output := buf.String()
	if !strings.Contains(output, "Removing 1 official package(s)...") {
		t.Errorf("Expected output to contain 'Removing 1 official package(s)...', got: %s", output)
	}
}

func TestRunRemove_DryRun(t *testing.T) {
	// Tests for dry-run logic integration if feasible,
	// or we verify dry-run is handled separately.
	// For now focusing on main execution path.
}
