package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	// Arrange: Set up test version info
	Version = "1.0.0"
	GitCommit = "abc123"
	BuildDate = "2024-12-31"

	// Act: Execute version command and capture output
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"version"})
	err := rootCmd.Execute()

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := buf.String()

	// Check that version info is present
	if !strings.Contains(output, "1.0.0") {
		t.Errorf("expected output to contain version '1.0.0', got: %s", output)
	}

	if !strings.Contains(output, "abc123") {
		t.Errorf("expected output to contain commit 'abc123', got: %s", output)
	}
}
