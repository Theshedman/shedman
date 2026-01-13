package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

func TestUpdateCmd_Flags(t *testing.T) {
	cmd := NewUpdateCmd()

	tests := []struct {
		name      string
		flag      string
		shorthand string
	}{
		{"yes", "yes", "y"},
		{"shedos", "shedos", ""},
		{"official", "official", ""},
		{"aur", "aur", ""},
		{"ignore", "ignore", ""},
		{"ignoregroup", "ignoregroup", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := cmd.Flags().Lookup(tt.flag)
			if f == nil {
				t.Fatalf("Flag --%s not found", tt.flag)
			}
			if tt.shorthand != "" && f.Shorthand != tt.shorthand {
				t.Errorf("Flag --%s shorthand got %s, want %s", tt.flag, f.Shorthand, tt.shorthand)
			}
		})
	}
}

func TestUpdateCmd_Run(t *testing.T) {
	// This test just ensures the command is runnable/callable
	// Logic testing might require mocking backends which is harder in this CLI structure
	cmd := NewUpdateCmd()

	if cmd.Use != "update [packages...]" {
		t.Errorf("Expected Use 'update [packages...]', got '%s'", cmd.Use)
	}
}

func TestRunUpdate(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	// Mock behavior
	syncCalled := false
	mock.SyncFunc = func() error {
		syncCalled = true
		return nil
	}

	upgradeCalled := false
	mock.UpgradeFunc = func(pkgs []string, options core.UpgradeOptions) error {
		upgradeCalled = true
		if len(pkgs) != 1 || pkgs[0] != "test-pkg" {
			t.Errorf("Expected upgrade of [test-pkg], got %v", pkgs)
		}
		if !options.NoConfirm {
			t.Error("Expected NoConfirm=true")
		}
		return nil
	}

	var buf bytes.Buffer
	pkgs := []string{"test-pkg"}
	opts := core.UpgradeOptions{
		NoConfirm: true,
	}

	// TDD: RunUpdate doesn't exist yet
	if err := RunUpdate(eng, &buf, pkgs, opts); err != nil {
		t.Fatalf("RunUpdate failed: %v", err)
	}

	if !syncCalled {
		t.Error("Backend.Sync was not called")
	}
	if !upgradeCalled {
		t.Error("Backend.Upgrade was not called")
	}

	output := buf.String()
	if !strings.Contains(output, "Starting full system upgrade") {
		t.Errorf("Expected output to contain 'Starting full system upgrade', got: %s", output)
	}
}
