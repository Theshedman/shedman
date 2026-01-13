package cmd

import (
	"bytes"
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
