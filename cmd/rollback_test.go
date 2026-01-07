package cmd

import (
	"testing"
)

func TestRollbackCmd_Flags(t *testing.T) {
	cmd := NewRollbackCmd()

	tests := []struct {
		name      string
		flag      string
		shorthand string
	}{
		{"list", "list", "l"},
		{"yes", "yes", "y"},
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

func TestRollbackCmd_Structure(t *testing.T) {
	cmd := NewRollbackCmd()
	if cmd.Use != "rollback <package>" {
		t.Errorf("Expected Use 'rollback <package>', got '%s'", cmd.Use)
	}
}
