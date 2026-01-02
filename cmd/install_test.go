package cmd

import (
	"testing"
)

func TestInstallCommand_Exists(t *testing.T) {
	installCmd := GetInstallCmd()
	if installCmd == nil {
		t.Fatal("Install command should exist")
	}

	if installCmd.Use != "install [packages...]" {
		t.Errorf("Expected Use 'install [packages...]', got '%s'", installCmd.Use)
	}
}

func TestInstallCommand_HasRequiredFlags(t *testing.T) {
	installCmd := GetInstallCmd()

	flags := []string{"needed", "asdeps", "asexplicit", "downloadonly", "aur", "official", "shedos", "overwrite"}

	for _, flag := range flags {
		if installCmd.Flags().Lookup(flag) == nil {
			t.Errorf("Missing flag: --%s", flag)
		}
	}
}

func TestInstallCommand_RequiresArgs(t *testing.T) {
	installCmd := GetInstallCmd()

	// Install command requires at least 1 package
	if installCmd.Args == nil {
		t.Error("Install command should have Args validation")
	}
}
