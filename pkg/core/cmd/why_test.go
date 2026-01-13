package cmd

import (
	"bytes"
	"fmt"
	"testing"
)

func TestRunWhy(t *testing.T) {
	tests := []struct {
		name      string
		pkg       string
		tree      bool // --tree flag
		lookError error
		runError  error
		wantError bool
		wantMsg   string
		wantArgs  []string // Expected args passed to pactree
	}{
		{
			name:      "Reverse Deps (Default)",
			pkg:       "foo",
			tree:      false,
			wantError: false,
			wantArgs:  []string{"-r", "-u", "foo"},
		},
		{
			name:      "Forward Deps (Tree)",
			pkg:       "foo",
			tree:      true,
			wantError: false,
			wantArgs:  []string{"-u", "foo"},
		},
		{
			name:      "Missing Pactree",
			pkg:       "foo",
			lookError: fmt.Errorf("not found"),
			wantError: true,
			wantMsg:   "pactree not found",
		},
		{
			name:      "Pactree Failed",
			pkg:       "foo",
			runError:  fmt.Errorf("exit status 1"),
			wantError: true,
			wantMsg:   "pactree failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := WhyDeps{
				LookPath: func(file string) (string, error) {
					return "/bin/pactree", tt.lookError
				},
				RunCmd: func(name string, args ...string) error {
					if name != "pactree" {
						t.Errorf("expected command pactree, got %s", name)
					}
					// Verify args
					if tt.wantArgs != nil {
						if len(args) != len(tt.wantArgs) {
							t.Errorf("args length mismatch: got %v, want %v", args, tt.wantArgs)
						} else {
							for i, arg := range args {
								if arg != tt.wantArgs[i] {
									t.Errorf("arg mismatch at %d: got %s, want %s", i, arg, tt.wantArgs[i])
								}
							}
						}
					}
					return tt.runError
				},
			}

			var buf bytes.Buffer
			err := RunWhy(deps, &buf, tt.pkg, tt.tree)
			if (err != nil) != tt.wantError {
				t.Errorf("RunWhy() error = %v, wantError %v", err, tt.wantError)
			}
			if err != nil && tt.wantMsg != "" {
				if len(err.Error()) < len(tt.wantMsg) {
					t.Errorf("error message '%s' does not contain '%s'", err.Error(), tt.wantMsg)
				}
			}
		})
	}
}
