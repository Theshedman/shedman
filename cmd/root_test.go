package cmd

import (
	"testing"
)

func TestGlobalFlagsRegistry(t *testing.T) {
	// Arrange
	flags := []string{"yes", "quiet", "verbose", "debug", "dry-run"}

	// Act & Assert
	for _, flagName := range flags {
		flag := rootCmd.PersistentFlags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected global flag --%s to be registered, but it was nil", flagName)
		}
	}
}

func TestGlobalFlagsParsing(t *testing.T) {
	// Arrange: Reset flags (variables are package-global)
	yesFlag = false
	quietFlag = false

	// Act: parse specific flags
	// Note: We use ParseFlags to test the flags logic without executing the command
	err := rootCmd.PersistentFlags().Parse([]string{"--yes", "--quiet"})
	if err != nil {
		t.Fatalf("unexpected error parsing flags: %v", err)
	}

	// Assert
	if !yesFlag {
		t.Error("expected yesFlag to be true after parsing --yes")
	}
	if !quietFlag {
		t.Error("expected quietFlag to be true after parsing --quiet")
	}
}
