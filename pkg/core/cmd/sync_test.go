package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

func TestSyncCommand_Exists(t *testing.T) {
	syncCmd := SyncCmd
	if syncCmd == nil {
		t.Fatal("Sync command should exist")
	}

	if syncCmd.Use != "sync" {
		t.Errorf("Expected Use 'sync', got '%s'", syncCmd.Use)
	}
}

func TestSyncCommand_HasRequiredFlags(t *testing.T) {
	syncCmd := SyncCmd

	flags := []string{"official", "aur", "shedos", "refresh", "debug", "dry-run", "quiet", "verbose"}

	for _, flag := range flags {
		if syncCmd.Flags().Lookup(flag) == nil {
			// Check local flags
			if flag == "official" || flag == "aur" || flag == "shedos" || flag == "refresh" {
				t.Errorf("Missing flag: --%s", flag)
			}
		}
	}
}

func TestSyncCommand_ShortDescription(t *testing.T) {
	syncCmd := SyncCmd

	if syncCmd.Short != "Sync package databases" {
		t.Errorf("Expected Short 'Sync package databases', got '%s'", syncCmd.Short)
	}
}

func TestRunSync(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	syncCalled := false
	mock.SyncFunc = func() error {
		syncCalled = true
		return nil
	}

	var buf bytes.Buffer
	// RunSync(eng, writer)
	if err := RunSync(eng, &buf); err != nil {
		t.Fatalf("RunSync failed: %v", err)
	}

	if !syncCalled {
		t.Error("Backend.Sync was not called")
	}

	output := buf.String()
	if !strings.Contains(output, "Synchronizing package databases") {
		t.Errorf("Expected output to contain 'Synchronizing package databases', got: %s", output)
	}
}
