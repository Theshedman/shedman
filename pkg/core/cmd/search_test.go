package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

func TestSearchCommand_Exists(t *testing.T) {
	searchCmd := SearchCmd
	if searchCmd == nil {
		t.Fatal("Search command should exist")
	}

	if searchCmd.Use != "search <query>" {
		t.Errorf("Expected Use 'search <query>', got '%s'", searchCmd.Use)
	}
}

func TestSearchCommand_HasRequiredFlags(t *testing.T) {
	searchCmd := SearchCmd

	flags := []string{"official", "aur", "shedos", "installed", "json", "limit"}

	for _, flag := range flags {
		if searchCmd.Flags().Lookup(flag) == nil {
			t.Errorf("Missing flag: --%s", flag)
		}
	}
}

func TestSearchCommand_RequiresArgs(t *testing.T) {
	searchCmd := SearchCmd

	// Search command requires query argument
	if searchCmd.Args == nil {
		t.Error("Search command should have Args validation")
	}
}

func TestSearchCommand_ShortDescription(t *testing.T) {
	searchCmd := SearchCmd

	if searchCmd.Short != "Search for packages" {
		t.Errorf("Expected Short 'Search for packages', got '%s'", searchCmd.Short)
	}
}

func TestRunSearch(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	mock.SearchFunc = func(query string) ([]core.PackageInfo, error) {
		if query == "neovim" {
			return []core.PackageInfo{
				{Name: "neovim", Version: "0.9.0", Description: "Editor", Source: core.SourceOfficial},
			}, nil
		}
		return []core.PackageInfo{}, nil
	}
	mock.IsInstalledFunc = func(name string) bool {
		return name == "neovim"
	}

	// Test Text Output
	var buf bytes.Buffer
	opts := SearchOptions{
		Limit: 10,
	}
	// RunSearch(eng, w, query, opts)
	if err := RunSearch(eng, &buf, "neovim", opts); err != nil {
		t.Fatalf("RunSearch text failed: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "neovim") || !strings.Contains(out, "0.9.0") {
		t.Errorf("Text output missing results. Got: %s", out)
	}
	if !strings.Contains(out, "Found 1 package(s)") {
		t.Errorf("Output missing summary. Got: %s", out)
	}

	// Validate JSON Output
	buf.Reset()
	opts.JSON = true
	if err := RunSearch(eng, &buf, "neovim", opts); err != nil {
		t.Fatalf("RunSearch json failed: %v", err)
	}
	out = buf.String()
	if !strings.Contains(out, "\"name\": \"neovim\"") {
		t.Errorf("JSON output invalid. Got: %s", out)
	}

	// Test No Results
	buf.Reset()
	if err := RunSearch(eng, &buf, "missing", opts); err != nil {
		// No error message expected for empty results, just log it
		t.Logf("Search for missing package returned error: %v", err)
	}

	out = buf.String()
	if !strings.Contains(out, "No packages found") && !strings.Contains(out, "[]") {
		t.Errorf("Expected 'No packages found' or empty json array, got: %s", out)
	}
}
