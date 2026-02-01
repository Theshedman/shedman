package cmd

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/pkg/core"
)

func TestRunClean(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	cfg := config.Default()

	tests := []struct {
		name      string
		opts      CleanOptions
		mockError error
		wantError bool
	}{
		{
			name:      "Clean Success (Cache Default)",
			opts:      CleanOptions{Cache: true},
			mockError: nil,
			wantError: false,
		},
		{
			name:      "Clean Keep 2",
			opts:      CleanOptions{Cache: true, Keep: 2},
			mockError: nil,
			wantError: false,
		},
		{
			name:      "Clean Error",
			opts:      CleanOptions{Cache: true},
			mockError: fmt.Errorf("failed"),
			wantError: true,
		},
		{
			name:      "Clean All Success",
			opts:      CleanOptions{All: true},
			mockError: nil,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock behavior
			cleaned := false
			mock.CleanCacheFunc = func(opts core.CleanOptions) error {
				if opts.All != tt.opts.All {
					t.Errorf("expected all=%v, got %v", tt.opts.All, opts.All)
				}
				if tt.opts.Keep > 0 && opts.Keep != tt.opts.Keep {
					t.Errorf("expected keep=%v, got %v", tt.opts.Keep, opts.Keep)
				}
				cleaned = true
				return tt.mockError
			}

			err := RunClean(context.Background(), eng, &bytes.Buffer{}, cfg, tt.opts)
			if (err != nil) != tt.wantError {
				t.Errorf("RunClean() error = %v, wantError %v", err, tt.wantError)
			}
			if !cleaned {
				t.Error("CleanCache was not called")
			}
		})
	}
}
