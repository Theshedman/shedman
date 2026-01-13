package cmd

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

func TestRunClean(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	tests := []struct {
		name      string
		all       bool
		keep      int
		mockError error
		wantError bool
	}{
		{
			name:      "Clean Success (Standard)",
			all:       false,
			keep:      0,
			mockError: nil,
			wantError: false,
		},
		{
			name:      "Clean Keep 2",
			all:       false,
			keep:      2,
			mockError: nil,
			wantError: false,
		},
		{
			name:      "Clean Error",
			all:       false,
			keep:      0,
			mockError: fmt.Errorf("failed"),
			wantError: true,
		},
		{
			name:      "Clean All Success",
			all:       true,
			keep:      0,
			mockError: nil,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock behavior
			cleaned := false
			mock.CleanCacheFunc = func(opts core.CleanOptions) error {
				if opts.All != tt.all {
					t.Errorf("expected all=%v, got %v", tt.all, opts.All)
				}
				if opts.Keep != tt.keep {
					t.Errorf("expected keep=%v, got %v", tt.keep, opts.Keep)
				}
				cleaned = true
				return tt.mockError
			}

			err := RunClean(eng, &bytes.Buffer{}, tt.all, tt.keep)
			if (err != nil) != tt.wantError {
				t.Errorf("RunClean() error = %v, wantError %v", err, tt.wantError)
			}
			if !cleaned {
				t.Error("CleanCache was not called")
			}
		})
	}
}
