package cmd

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

func TestRunOrphans(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	tests := []struct {
		name        string
		orphans     []string
		remove      bool
		listError   error
		removeError error
		wantError   bool
	}{
		{
			name:      "No Orphans",
			orphans:   []string{},
			remove:    false,
			wantError: false,
		},
		{
			name:      "List Error",
			listError: fmt.Errorf("list fail"),
			wantError: true,
		},
		{
			name:      "List Only",
			orphans:   []string{"pkg1", "pkg2"},
			remove:    false,
			wantError: false,
		},
		{
			name:      "Remove Success",
			orphans:   []string{"pkg1"},
			remove:    true,
			wantError: false,
		},
		{
			name:        "Remove Error",
			orphans:     []string{"pkg1"},
			remove:      true,
			removeError: fmt.Errorf("remove fail"),
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.ListOrphansFunc = func() ([]string, error) {
				return tt.orphans, tt.listError
			}
			calledRemove := false
			mock.RemoveOrphansFunc = func(pkgs []string) error {
				calledRemove = true
				if len(pkgs) != len(tt.orphans) {
					t.Errorf("got %d orphans to remove, want %d", len(pkgs), len(tt.orphans))
				}
				return tt.removeError
			}

			var buf bytes.Buffer
			err := RunOrphans(eng, &buf, tt.remove)
			if (err != nil) != tt.wantError {
				t.Errorf("RunOrphans() error = %v, wantError %v", err, tt.wantError)
			}
			if tt.remove && len(tt.orphans) > 0 && tt.listError == nil {
				if !calledRemove {
					t.Error("RemoveOrphans should be called")
				}
			}
			// Add basic output assertions
			if !tt.wantError && len(tt.orphans) > 0 {
				if tt.remove {
					if !strings.Contains(buf.String(), "Orphans removed") {
						t.Errorf("Expected success msg, got: %s", buf.String())
					}
				} else {
					if !strings.Contains(buf.String(), "Use --remove") {
						t.Errorf("Expected usage hint, got: %s", buf.String())
					}
				}
			}
		})
	}
}
