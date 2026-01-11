package commands

import (
	"bytes"
	"strings"
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

func TestInstallCommand_HelpOutput(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"install", "--help"})
	_ = rootCmd.Execute()

	output := buf.String()

	// Verify help includes key information
	expected := []string{
		"Install packages",
		"neovim@0.10.0",
		"--needed",
		"--aur",
		"--official",
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("Expected help to contain '%s'", exp)
		}
	}
}

func TestInstallCommand_ShortDescription(t *testing.T) {
	installCmd := GetInstallCmd()

	if installCmd.Short != "Install packages" {
		t.Errorf("Expected Short 'Install packages', got '%s'", installCmd.Short)
	}
}
