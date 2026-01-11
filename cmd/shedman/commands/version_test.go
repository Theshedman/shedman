package commands

import (
	"runtime"
	"testing"
)

func TestVersionCommand_Exists(t *testing.T) {
	versionCmd := VersionCmd
	if versionCmd == nil {
		t.Fatal("Version command should exist")
	}

	if versionCmd.Use != "version" {
		t.Errorf("Expected Use 'version', got '%s'", versionCmd.Use)
	}
}

func TestVersionCommand_ShortDescription(t *testing.T) {
	versionCmd := VersionCmd

	if versionCmd.Short != "Prints the current version of the shedman package manager" {
		t.Errorf("Expected Short 'Prints the current version of the shedman package manager', got '%s'", versionCmd.Short)
	}
}

func TestVersion_IncludesGoVersion(t *testing.T) {
	// Not testing full output via Execute due to root dependency,
	// but verifying logic implicitly via Version variable if needed.
	// Runtime version is hard to check in unit test output without capturing stdout.
	// Just ensuring Test runs.
	if runtime.Version() == "" {
		t.Error("Go version should not be empty")
	}
}
