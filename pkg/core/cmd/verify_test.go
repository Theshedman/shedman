package cmd

import (
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

func TestRunVerify(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	tests := []struct {
		name          string
		args          []string
		fix           bool
		pkg           string // Expected package to be verified (empty for all)
		issues        []string
		verifyAllErr  error
		mockError     error
		wantError     bool
		wantReinstall bool
	}{
		{
			name:      "Healthy Package",
			args:      []string{"foo"},
			pkg:       "foo",
			issues:    nil,
			wantError: false,
		},
		{
			name:      "Corrupted Package",
			args:      []string{"foo"},
			pkg:       "foo",
			issues:    []string{"/usr/bin/foo: size mismatch"},
			wantError: false,
		},
		{
			name:          "Corrupted Package Fix",
			args:          []string{"foo"},
			fix:           true,
			pkg:           "foo",
			issues:        []string{"/usr/bin/foo: size mismatch"},
			wantError:     false,
			wantReinstall: true,
		},
		{
			name:      "Verify All Success",
			args:      []string{},
			wantError: false,
		},
		{
			name:         "Verify All With Issues",
			args:         []string{},
			verifyAllErr: nil, // Mock returns map of issues
			wantError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reinstallCalled := false

			// Mock VerifyPackage
			mock.VerifyPackageFunc = func(pkgName string) ([]string, error) {
				if pkgName != tt.pkg {
					t.Errorf("expected verify check for %s, got %s", tt.pkg, pkgName)
				}
				return tt.issues, tt.mockError
			}

			// Mock VerifyAll
			mock.VerifyAllFunc = func() (map[string][]string, error) {
				if len(tt.issues) > 0 {
					return map[string][]string{"broken-pkg": tt.issues}, nil
				}
				return nil, tt.verifyAllErr
			}

			// Mock Install for fix
			mock.InstallFunc = func(pkgs []string, opts core.InstallOptions) error {
				reinstallCalled = true
				if len(pkgs) != 1 || pkgs[0] != tt.pkg {
					t.Errorf("expected reinstall of %s, got %v", tt.pkg, pkgs)
				}
				return nil
			}

			err := RunVerify(eng, tt.args, tt.fix)
			if (err != nil) != tt.wantError {
				t.Errorf("RunVerify() error = %v, wantError %v", err, tt.wantError)
			}

			if tt.wantReinstall && !reinstallCalled {
				t.Error("Expected reinstall to be called, but it was not")
			}
		})
	}
}
