package main

import (
	"testing"
)

func TestGlobalFlagsRegistry(t *testing.T) {
	// All global flags from docs/README.md
	flags := []string{
		"yes", "noconfirm", "quiet", "verbose",
		"debug", "dry-run", "color", "no-color", "config",
	}

	for _, flagName := range flags {
		flag := rootCmd.PersistentFlags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected global flag --%s to be registered, but it was nil", flagName)
		}
	}
}

func TestGlobalFlagsParsing(t *testing.T) {
	// Reset flags
	yesFlag = false
	quietFlag = false
	noconfirmFlag = false

	err := rootCmd.PersistentFlags().Parse([]string{"--yes", "--quiet", "--noconfirm"})
	if err != nil {
		t.Fatalf("unexpected error parsing flags: %v", err)
	}

	if !yesFlag {
		t.Error("expected yesFlag to be true after parsing --yes")
	}
	if !quietFlag {
		t.Error("expected quietFlag to be true after parsing --quiet")
	}
	if !noconfirmFlag {
		t.Error("expected noconfirmFlag to be true after parsing --noconfirm")
	}
}

func TestNoconfirmIsAliasForYes(t *testing.T) {
	// Reset flags
	yesFlag = false
	noconfirmFlag = true

	// PersistentPreRun should set yesFlag when noconfirmFlag is true
	rootCmd.PersistentPreRun(rootCmd, []string{})

	if !yesFlag {
		t.Error("expected yesFlag to be true when noconfirmFlag is true")
	}
}
