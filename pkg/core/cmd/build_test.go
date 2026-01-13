package cmd

import (
	"fmt"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

func TestRunBuild(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	tests := []struct {
		name      string
		dir       string
		opts      core.BuildOptions
		mockError error
		wantError bool
	}{
		{
			name:      "Build Success",
			dir:       "/tmp/build",
			opts:      core.BuildOptions{Clean: true, Install: true, SynDeps: true, NoConfirm: true},
			wantError: false,
		},
		{
			name:      "Build Failure",
			dir:       ".",
			mockError: fmt.Errorf("makepkg failed"),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.BuildFunc = func(dir string, opts core.BuildOptions) error {
				if dir != tt.dir {
					t.Errorf("expected dir=%s, got %s", tt.dir, dir)
				}
				if opts != tt.opts {
					t.Errorf("expected opts=%v, got %v", tt.opts, opts)
				}
				return tt.mockError
			}

			err := RunBuild(eng, tt.dir, tt.opts)
			if (err != nil) != tt.wantError {
				t.Errorf("RunBuild() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
