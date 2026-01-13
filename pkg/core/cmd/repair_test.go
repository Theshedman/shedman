package cmd

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

func TestRunRepair(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	tests := []struct {
		name      string
		action    string
		mockError error
		wantError bool
	}{
		{
			name:      "Remove Lock Success",
			action:    "lock",
			wantError: false,
		},
		{
			name:      "Remove Lock Fail",
			action:    "lock",
			mockError: fmt.Errorf("failed to remove lock"),
			wantError: true,
		},
		{
			name:      "Unknown Action",
			action:    "unknown",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.RemoveLockFunc = func() error { return tt.mockError }

			var buf bytes.Buffer
			err := RunRepair(eng, &buf, tt.action)
			if (err != nil) != tt.wantError {
				t.Errorf("RunRepair(%s) error = %v, wantError %v", tt.action, err, tt.wantError)
			}
		})
	}
}
