package output_test

import (
	"testing"
	"time"

	"github.com/theshedman/shedman/internal/output"
)

func TestConfirmOptions_SkipPrompt(t *testing.T) {
	// When SkipPrompt is true, should return Default without reading stdin
	opts := output.ConfirmOptions{
		Default:    true,
		SkipPrompt: true,
	}

	// This should not block since SkipPrompt is true
	result := output.Confirm("Test?", opts)
	if !result {
		t.Error("Expected true when SkipPrompt=true and Default=true")
	}

	opts.Default = false
	result = output.Confirm("Test?", opts)
	if result {
		t.Error("Expected false when SkipPrompt=true and Default=false")
	}
}

func TestConfirmInstall_Skip(t *testing.T) {
	result := output.ConfirmInstall(true)
	if !result {
		t.Error("ConfirmInstall with skip should return true")
	}
}

func TestConfirmRemoval_Skip(t *testing.T) {
	result := output.ConfirmRemoval([]string{"pkg1", "pkg2"}, true)
	if result {
		t.Error("ConfirmRemoval with skip should return false (default)")
	}
}

func TestSummaryLine(t *testing.T) {
	line := output.SummaryLine{
		Label: "Test",
		Value: "123",
	}
	if line.Label != "Test" || line.Value != "123" {
		t.Error("SummaryLine fields not set correctly")
	}
}

func TestConfirmOptions_Timeout(t *testing.T) {
	// Test that Timeout field exists and can be set
	opts := output.ConfirmOptions{
		Default: true,
		Timeout: 5 * time.Second,
	}
	if opts.Timeout != 5*time.Second {
		t.Error("Timeout field not set correctly")
	}
}
