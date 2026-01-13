package cmd

import (
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

func TestRunSize(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	tests := []struct {
		name      string
		pkg       string
		info      *core.PackageInfo
		mockError error
		wantError bool
	}{
		{
			name: "Success",
			pkg:  "foo",
			info: &core.PackageInfo{
				Name:          "foo",
				InstalledSize: 1024 * 1024 * 5, // 5 MB
				Size:          1024 * 1024,     // 1 MB
			},
			wantError: false,
		},
		{
			name:      "Not Found",
			pkg:       "bar",
			info:      nil,
			mockError: core.ErrPackageNotFound,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.InfoFunc = func(pkgName string) (*core.PackageInfo, error) {
				if pkgName != tt.pkg {
					t.Errorf("expected info for %s, got %s", tt.pkg, pkgName)
				}
				return tt.info, tt.mockError
			}

			err := RunSize(eng, tt.pkg)
			if (err != nil) != tt.wantError {
				t.Errorf("RunSize() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
