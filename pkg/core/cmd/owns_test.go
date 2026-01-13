package cmd

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

func TestRunOwns(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	tests := []struct {
		name      string
		path      string
		owner     string
		mockError error
		wantError bool
		wantOut   string
	}{
		{
			name:      "Owned",
			path:      "/bin/ls",
			owner:     "coreutils",
			wantError: false,
			wantOut:   "owned by 'coreutils'",
		},
		{
			name:      "Not Owned",
			path:      "/tmp/foo",
			mockError: fmt.Errorf("no package owns"),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.GetFileOwnerFunc = func(path string) (string, error) {
				if path != tt.path {
					t.Errorf("expected path=%s, got %s", tt.path, path)
				}
				return tt.owner, tt.mockError
			}

			var buf bytes.Buffer
			// RunOwns(eng, w, path)
			err := RunOwns(eng, &buf, tt.path)
			if (err != nil) != tt.wantError {
				t.Errorf("RunOwns() error = %v, wantError %v", err, tt.wantError)
			}

			if !tt.wantError {
				if !strings.Contains(buf.String(), tt.wantOut) {
					t.Errorf("Output missing '%s'. Got: %s", tt.wantOut, buf.String())
				}
			}
		})
	}
}
