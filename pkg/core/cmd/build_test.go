package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

func TestRunBuild(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	buildCalled := false
	var capturedDir string
	var capturedOpts core.BuildOptions

	mock.BuildFunc = func(dir string, opts core.BuildOptions) error {
		buildCalled = true
		capturedDir = dir
		capturedOpts = opts
		return nil
	}

	opts := core.BuildOptions{
		Clean: true,
	}

	var buf bytes.Buffer
	if err := RunBuild(eng, &buf, "/tmp/pkg", opts); err != nil {
		t.Fatalf("RunBuild failed: %v", err)
	}

	if !buildCalled {
		t.Error("Build was not called")
	}
	if capturedDir != "/tmp/pkg" {
		t.Errorf("Expected dir '/tmp/pkg', got '%s'", capturedDir)
	}
	if !capturedOpts.Clean {
		t.Error("Expected Clean=true")
	}
	if !strings.Contains(buf.String(), "Building package in /tmp/pkg") {
		t.Errorf("Unexpected output: %s", buf.String())
	}
}

func TestEditPKGBUILD(t *testing.T) {
	tmpDir := t.TempDir()
	pkgbuild := filepath.Join(tmpDir, "PKGBUILD")
	if err := os.WriteFile(pkgbuild, []byte("pkgname=test\n"), 0644); err != nil {
		t.Fatalf("write PKGBUILD: %v", err)
	}

	origRunner := editorRunner
	called := false
	editorRunner = func(_ string, args []string) error {
		called = true
		if len(args) == 0 || args[len(args)-1] != pkgbuild {
			t.Errorf("expected PKGBUILD path in args, got %v", args)
		}
		return nil
	}
	t.Cleanup(func() { editorRunner = origRunner })

	if err := editPKGBUILD(tmpDir, "vim"); err != nil {
		t.Fatalf("editPKGBUILD failed: %v", err)
	}
	if !called {
		t.Error("expected editor runner to be called")
	}
}
