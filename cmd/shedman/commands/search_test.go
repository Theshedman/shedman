package commands

import (
	"testing"
)

func TestSearchCommand_Exists(t *testing.T) {
	searchCmd := SearchCmd
	if searchCmd == nil {
		t.Fatal("Search command should exist")
	}

	if searchCmd.Use != "search <query>" {
		t.Errorf("Expected Use 'search <query>', got '%s'", searchCmd.Use)
	}
}

func TestSearchCommand_HasRequiredFlags(t *testing.T) {
	searchCmd := SearchCmd

	flags := []string{"official", "aur", "shedos", "installed", "json", "limit"}

	for _, flag := range flags {
		if searchCmd.Flags().Lookup(flag) == nil {
			t.Errorf("Missing flag: --%s", flag)
		}
	}
}

func TestSearchCommand_RequiresArgs(t *testing.T) {
	searchCmd := SearchCmd

	// Search command requires query argument
	if searchCmd.Args == nil {
		t.Error("Search command should have Args validation")
	}
}

func TestSearchCommand_ShortDescription(t *testing.T) {
	searchCmd := SearchCmd

	if searchCmd.Short != "Search for packages" {
		t.Errorf("Expected Short 'Search for packages', got '%s'", searchCmd.Short)
	}
}
