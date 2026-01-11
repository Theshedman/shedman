package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestSyncCommand(t *testing.T) {
	// Arrange
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	// Reset flags
	quietFlag = false
	verboseFlag = false

	rootCmd.SetArgs([]string{"sync", "--help"})

	// Act
	err := rootCmd.Execute()

	// Assert
	if err != nil {
		t.Fatalf("expected no error, but got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Synchronize package databases") {
		t.Errorf("expected help to contain description, got: %s", output)
	}
}

func TestSyncCommand_Flags(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"sync", "--help"})
	_ = rootCmd.Execute()

	output := buf.String()

	// Verify all flags are documented
	flags := []string{"--official", "--aur", "--shedos", "--refresh"}
	for _, flag := range flags {
		if !strings.Contains(output, flag) {
			t.Errorf("expected help to contain %s flag", flag)
		}
	}
}

// TestSyncCommand_HasForceFlag verifies --force flag exists as alias for --refresh
func TestSyncCommand_HasForceFlag(t *testing.T) {
	if syncCmd.Flags().Lookup("force") == nil {
		t.Error("expected sync command to have --force flag")
	}
}

// Note: The following tests require mock backends which we'll add to the shedman package.
// For now, these tests verify the command structure.

// TestSyncCommand_RequiresNoArgs verifies sync works with no arguments
func TestSyncCommand_RequiresNoArgs(t *testing.T) {
	// sync command should accept zero arguments (syncs all by default)
	if syncCmd.Args != nil {
		// If Args is set, it should allow zero args
		err := syncCmd.Args(syncCmd, []string{})
		if err != nil {
			t.Errorf("sync should accept zero arguments, got error: %v", err)
		}
	}
}
