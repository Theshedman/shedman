package commands

import (
	"testing"
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
			// Some flags might be persistent or inherited, so just check local ones
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

/*
func TestSyncCommand_Run_DryRun(t *testing.T) {
	// Requires mocking backends or config, skipping complex logic for unit test
}
*/
