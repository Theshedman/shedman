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
	flags := []string{"--official", "--aur", "--shedos"}
	for _, flag := range flags {
		if !strings.Contains(output, flag) {
			t.Errorf("expected help to contain %s flag", flag)
		}
	}
}
