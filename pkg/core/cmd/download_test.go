package cmd

import (
	"fmt"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

func TestRunDownload(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	tests := []struct {
		name           string
		pkgs           []string
		mockError      error
		wantError      bool
		wantDownloaded bool
	}{
		{
			name:           "Success",
			pkgs:           []string{"foo"},
			wantError:      false,
			wantDownloaded: true,
		},
		{
			name:           "Fail",
			pkgs:           []string{"foo"},
			mockError:      fmt.Errorf("failed"),
			wantError:      true,
			wantDownloaded: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			downloadCalled := false
			mock.InstallFunc = func(pkgs []string, opts core.InstallOptions) error {
				if !opts.DownloadOnly {
					t.Error("Expected DownloadOnly to be true")
				}
				if len(pkgs) != len(tt.pkgs) {
					t.Errorf("pkg count mismatch")
				}
				downloadCalled = true
				return tt.mockError
			}

			err := RunDownload(eng, tt.pkgs)
			if (err != nil) != tt.wantError {
				t.Errorf("RunDownload() error = %v, wantError %v", err, tt.wantError)
			}
			if tt.wantDownloaded && !downloadCalled {
				t.Error("Install was not called")
			}
		})
	}
}
