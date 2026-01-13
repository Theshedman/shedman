package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

func TestRunImport(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	var installedPkgs []string
	mock.InstallFunc = func(pkgs []string, opts core.InstallOptions) error {
		installedPkgs = pkgs
		return nil
	}

	input := `pkg1
pkg2

# comment
pkg3`
	r := strings.NewReader(input)
	var w bytes.Buffer

	err := RunImport(eng, &w, r)
	if err != nil {
		t.Fatalf("RunImport failed: %v", err)
	}

	if len(installedPkgs) != 3 {
		t.Errorf("Expected 3 packages, got %d", len(installedPkgs))
	}
	// Check order or content
	has := func(p string) bool {
		for _, v := range installedPkgs {
			if v == p {
				return true
			}
		}
		return false
	}
	if !has("pkg1") || !has("pkg2") || !has("pkg3") {
		t.Errorf("Missing packages in install call: %v", installedPkgs)
	}
}

func TestRunImport_Empty(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	mock.InstallFunc = func(pkgs []string, opts core.InstallOptions) error {
		t.Error("Install should not be called for empty input")
		return nil
	}

	input := `
# just comments
`
	r := strings.NewReader(input)
	var w bytes.Buffer

	if err := RunImport(eng, &w, r); err != nil {
		t.Fatalf("RunImport failed: %v", err)
	}
}
