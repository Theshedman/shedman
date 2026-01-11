package commands

import (
	"bytes"
	"strings"
	"testing"
)

func TestSearchCommand_Exists(t *testing.T) {
	searchCmd := GetSearchCmd()
	if searchCmd == nil {
		t.Fatal("Search command should exist")
	}

	if searchCmd.Use != "search <query>" {
		t.Errorf("Expected Use 'search <query>', got '%s'", searchCmd.Use)
	}
}

func TestSearchCommand_HasRequiredFlags(t *testing.T) {
	searchCmd := GetSearchCmd()

	flags := []string{"official", "aur", "shedos", "installed", "json", "limit"}

	for _, flag := range flags {
		if searchCmd.Flags().Lookup(flag) == nil {
			t.Errorf("Missing flag: --%s", flag)
		}
	}
}

func TestSearchCommand_RequiresArgs(t *testing.T) {
	searchCmd := GetSearchCmd()

	// Search command requires at least 1 argument (query)
	if searchCmd.Args == nil {
		t.Error("Search command should have Args validation")
	}
}

func TestSearchCommand_HelpOutput(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"search", "--help"})
	_ = rootCmd.Execute()

	output := buf.String()

	// Verify help includes key information
	expected := []string{
		"Search for packages",
		"--official",
		"--aur",
		"--shedos",
		"--installed",
		"--json",
		"--limit",
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("Expected help to contain '%s'", exp)
		}
	}
}

func TestSearchCommand_ShortDescription(t *testing.T) {
	searchCmd := GetSearchCmd()

	if searchCmd.Short != "Search for packages" {
		t.Errorf("Expected Short 'Search for packages', got '%s'", searchCmd.Short)
	}
}
