package output_test

import (
"testing"

"github.com/theshedman/shedman/pkg/shedman/output"
)

func TestColorize_WhenEnabled(t *testing.T) {
	output.SetColorEnabled(true)
	result := output.Colorize(output.Green, "test")
	if result == "test" {
		t.Error("Colorize should add color codes when enabled")
	}
	if result != output.Green+"test"+output.Reset {
		t.Errorf("Unexpected result: %s", result)
	}
}

func TestColorize_WhenDisabled(t *testing.T) {
	output.SetColorEnabled(false)
	result := output.Colorize(output.Green, "test")
	if result != "test" {
		t.Error("Colorize should return plain text when disabled")
	}
}

func TestInitColor_NoColor(t *testing.T) {
	output.InitColor(false, true)
	if output.IsColorEnabled() {
		t.Error("Color should be disabled with --no-color")
	}
}

func TestInitColor_ForceColor(t *testing.T) {
	output.InitColor(true, false)
	if !output.IsColorEnabled() {
		t.Error("Color should be enabled with --color")
	}
}
