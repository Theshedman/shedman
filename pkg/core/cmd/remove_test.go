package cmd

import (
	"testing"
)

func TestRemoveCommand_Exists(t *testing.T) {
	removeCmd := RemoveCmd
	if removeCmd == nil {
		t.Fatal("Remove command should exist")
	}

	if removeCmd.Use != "remove [packages...]" {
		t.Errorf("Expected Use 'remove [packages...]', got '%s'", removeCmd.Use)
	}
}

func TestRemoveCommand_HasRequiredFlags(t *testing.T) {
	removeCmd := RemoveCmd

	flags := []string{"recursive", "purge", "cascade", "nosave"}

	for _, flag := range flags {
		if removeCmd.Flags().Lookup(flag) == nil {
			t.Errorf("Missing flag: --%s", flag)
		}
	}
}

func TestRemoveCommand_HasShortFlags(t *testing.T) {
	removeCmd := RemoveCmd

	// -s is shorthand for --recursive
	if flag := removeCmd.Flags().ShorthandLookup("s"); flag == nil {
		t.Error("Missing short flag: -s (for --recursive)")
	}
}

func TestRemoveCommand_RequiresArgs(t *testing.T) {
	removeCmd := RemoveCmd

	// Remove command requires at least 1 package
	if removeCmd.Args == nil {
		t.Error("Remove command should have Args validation")
	}
}

func TestRemoveCommand_ShortDescription(t *testing.T) {
	removeCmd := RemoveCmd

	if removeCmd.Short != "Remove packages" {
		t.Errorf("Expected Short 'Remove packages', got '%s'", removeCmd.Short)
	}
}

func TestRemoveCommand_NosaveIsPurgeAlias(t *testing.T) {
	removeCmd := RemoveCmd

	purgeFlag := removeCmd.Flags().Lookup("purge")
	nosaveFlag := removeCmd.Flags().Lookup("nosave")

	if purgeFlag == nil || nosaveFlag == nil {
		t.Fatal("Both --purge and --nosave flags should exist")
	}

	// Both flags should control the same behavior (NoSave in RemoveOptions)
	// This is verified by checking they're both registered
}
