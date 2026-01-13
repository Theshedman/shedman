package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

func TestInfoCmd_Flags(t *testing.T) {
	cmd := NewInfoCmd()

	tests := []struct {
		name      string
		flag      string
		shorthand string
	}{
		{"json", "json", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := cmd.Flags().Lookup(tt.flag)
			if f == nil {
				t.Fatalf("Flag --%s not found", tt.flag)
			}
			if tt.shorthand != "" && f.Shorthand != tt.shorthand {
				t.Errorf("Flag --%s shorthand got %s, want %s", tt.flag, f.Shorthand, tt.shorthand)
			}
		})
	}
}

func TestInfoCmd_Structure(t *testing.T) {
	cmd := NewInfoCmd()
	if cmd.Use != "info [package]" {
		t.Errorf("Expected Use 'info [package]', got '%s'", cmd.Use)
	}

	// Check Args validation if possible, usually requires execution
}

func TestRunInfo(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	mock.InfoFunc = func(pkgName string) (*core.PackageInfo, error) {
		if pkgName != "neovim" {
			return nil, core.ErrPackageNotFound
		}
		return &core.PackageInfo{
			Name:        "neovim",
			Version:     "0.9.0",
			Description: "Vim-fork focused on extensibility",
			Source:      core.SourceOfficial,
			DownloadURL: "https://example.com",
		}, nil
	}

	// Test Text Output
	var buf bytes.Buffer
	if err := RunInfo(eng, &buf, "neovim", false); err != nil {
		t.Fatalf("RunInfo text failed: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "neovim") || !strings.Contains(out, "0.9.0") {
		t.Errorf("Text output missing fields. Got: %s", out)
	}

	// Test JSON Output
	buf.Reset()
	if err := RunInfo(eng, &buf, "neovim", true); err != nil {
		t.Fatalf("RunInfo json failed: %v", err)
	}
	out = buf.String()
	if !strings.Contains(out, "{") || !strings.Contains(out, "\"Name\": \"neovim\"") {
		t.Errorf("JSON output invalid. Got: %s", out)
	}

	// Test Not Found
	if err := RunInfo(eng, &buf, "missing", false); err == nil {
		t.Error("Expected error for missing package")
	}
}
