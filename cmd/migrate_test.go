package cmd

import (
	"strings"
	"testing"
)

func TestMigrateCmd_Exists(t *testing.T) {
	if migrateCmd == nil {
		t.Fatal("migrateCmd should not be nil")
	}

	if migrateCmd.Use != "migrate" {
		t.Errorf("Expected Use 'migrate', got %s", migrateCmd.Use)
	}

	if migrateCmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if migrateCmd.Long == "" {
		t.Error("Long description should not be empty")
	}
}

func TestMigrateCmd_Flags(t *testing.T) {
	tests := []struct {
		name string
		flag string
	}{
		{"pacman", "pacman"},
		{"apt", "apt"},
		{"dnf", "dnf"},
		{"auto", "auto"},
		{"dry-run", "dry-run"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := migrateCmd.Flags().Lookup(tt.flag)
			if flag == nil {
				t.Errorf("Flag --%s should exist", tt.flag)
			}
		})
	}
}

func TestMigrateCmd_RequiresSource(t *testing.T) {
	// Save original values
	origPacman := migrateFromPacman
	origApt := migrateFromApt
	origDnf := migrateFromDnf
	origAuto := migrateAuto

	// Cleanup after test
	defer func() {
		migrateFromPacman = origPacman
		migrateFromApt = origApt
		migrateFromDnf = origDnf
		migrateAuto = origAuto
	}()

	// Reset flags
	migrateFromPacman = ""
	migrateFromApt = ""
	migrateFromDnf = ""
	migrateAuto = false

	err := migrateCmd.RunE(migrateCmd, []string{})
	if err == nil {
		t.Fatal("Expected error when no source specified")
	}

	expected := "please specify a source: --pacman, --apt, --dnf, or --auto"
	if err.Error() != expected {
		t.Errorf("Expected error %q, got %q", expected, err.Error())
	}
}

func TestMigrateCmd_AptNotImplemented(t *testing.T) {
	// Save original values
	origPacman := migrateFromPacman
	origApt := migrateFromApt
	origDnf := migrateFromDnf
	origAuto := migrateAuto

	// Cleanup after test
	defer func() {
		migrateFromPacman = origPacman
		migrateFromApt = origApt
		migrateFromDnf = origDnf
		migrateAuto = origAuto
	}()

	migrateFromPacman = ""
	migrateFromApt = "true"
	migrateFromDnf = ""
	migrateAuto = false

	err := migrateCmd.RunE(migrateCmd, []string{})
	if err == nil {
		t.Fatal("Expected error for apt migration")
	}

	if !strings.Contains(err.Error(), "apt migration not yet implemented") {
		t.Errorf("Expected apt not implemented error, got: %v", err)
	}
}

func TestMigrateCmd_DnfNotImplemented(t *testing.T) {
	// Save original values
	origPacman := migrateFromPacman
	origApt := migrateFromApt
	origDnf := migrateFromDnf
	origAuto := migrateAuto

	// Cleanup after test
	defer func() {
		migrateFromPacman = origPacman
		migrateFromApt = origApt
		migrateFromDnf = origDnf
		migrateAuto = origAuto
	}()

	migrateFromPacman = ""
	migrateFromApt = ""
	migrateFromDnf = "true"
	migrateAuto = false

	err := migrateCmd.RunE(migrateCmd, []string{})
	if err == nil {
		t.Fatal("Expected error for dnf migration")
	}

	if !strings.Contains(err.Error(), "dnf migration not yet implemented") {
		t.Errorf("Expected dnf not implemented error, got: %v", err)
	}
}

func TestMigrateCmd_InvalidPacmanPath(t *testing.T) {
	// Save original values
	origPacman := migrateFromPacman
	origApt := migrateFromApt
	origDnf := migrateFromDnf
	origAuto := migrateAuto

	// Cleanup after test
	defer func() {
		migrateFromPacman = origPacman
		migrateFromApt = origApt
		migrateFromDnf = origDnf
		migrateAuto = origAuto
	}()

	migrateFromPacman = "/nonexistent/path/pacman.conf"
	migrateFromApt = ""
	migrateFromDnf = ""
	migrateAuto = false

	err := migrateCmd.RunE(migrateCmd, []string{})
	if err == nil {
		t.Fatal("Expected error for invalid pacman path")
	}

	// Should fail to parse the nonexistent file
	if !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("Expected parse error, got: %v", err)
	}
}

func TestDetectDistro(t *testing.T) {
	// This test ensures the function doesn't panic and returns valid types
	pm, path := detectDistro()

	// Validate return types are strings
	_ = pm + path

	// If detected, both should be non-empty
	if pm != "" && path == "" {
		t.Error("If package manager detected, path should not be empty")
	}

	t.Logf("Detected: pm=%q, path=%q", pm, path)
}
