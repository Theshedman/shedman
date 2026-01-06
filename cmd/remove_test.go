package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRemoveCommand_Exists(t *testing.T) {
	removeCmd := GetRemoveCmd()
	if removeCmd == nil {
		t.Fatal("Remove command should exist")
	}

	if removeCmd.Use != "remove [packages...]" {
		t.Errorf("Expected Use 'remove [packages...]', got '%s'", removeCmd.Use)
	}
}

func TestRemoveCommand_HasRequiredFlags(t *testing.T) {
	removeCmd := GetRemoveCmd()

	flags := []string{"recursive", "purge"}

	for _, flag := range flags {
		if removeCmd.Flags().Lookup(flag) == nil {
			t.Errorf("Missing flag: --%s", flag)
		}
	}
}

func TestRemoveCommand_HasShortFlags(t *testing.T) {
	removeCmd := GetRemoveCmd()

	// -s is shorthand for --recursive
	flag := removeCmd.Flags().ShorthandLookup("s")
	if flag == nil {
		t.Error("Missing short flag: -s (for --recursive)")
	}
}

func TestRemoveCommand_RequiresArgs(t *testing.T) {
	removeCmd := GetRemoveCmd()

	// Remove command requires at least 1 package
	if removeCmd.Args == nil {
		t.Error("Remove command should have Args validation")
	}
}

func TestRemoveCommand_HelpOutput(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"remove", "--help"})
	_ = rootCmd.Execute()

	output := buf.String()

	// Verify help includes key information
	expected := []string{
		"Remove installed packages",
		"--recursive",
		"--purge",
		"-s",
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("Expected help to contain '%s'", exp)
		}
	}
}

func TestRemoveCommand_ShortDescription(t *testing.T) {
	removeCmd := GetRemoveCmd()

	if removeCmd.Short != "Remove packages" {
		t.Errorf("Expected Short 'Remove packages', got '%s'", removeCmd.Short)
	}
}
