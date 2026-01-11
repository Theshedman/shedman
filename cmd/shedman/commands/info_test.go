package commands

import (
	"testing"
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
