package commands

import (
	"testing"
)

func TestUpdateCmd_Flags(t *testing.T) {
	cmd := NewUpdateCmd()

	tests := []struct {
		name      string
		flag      string
		shorthand string
	}{
		{"yes", "yes", "y"},
		{"shedos", "shedos", ""},
		{"official", "official", ""},
		{"aur", "aur", ""},
		{"ignore", "ignore", ""},
		{"ignoregroup", "ignoregroup", ""},
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

func TestUpdateCmd_Run(t *testing.T) {
	// This test just ensures the command is runnable/callable
	// Logic testing might require mocking backends which is harder in this CLI structure
	cmd := NewUpdateCmd()
	if cmd.Use != "update" {
		t.Errorf("Expected Use 'update', got '%s'", cmd.Use)
	}
}
