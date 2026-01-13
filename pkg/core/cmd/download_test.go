package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

func TestRunDownload(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	installCalled := false
	var capturedPkgs []string
	var capturedOpts core.InstallOptions

	mock.InstallFunc = func(pkgs []string, opts core.InstallOptions) error {
		installCalled = true
		capturedPkgs = pkgs
		capturedOpts = opts
		return nil
	}

	pkgs := []string{"foo", "bar"}
	var buf bytes.Buffer
	if err := RunDownload(eng, &buf, pkgs); err != nil {
		t.Fatalf("RunDownload failed: %v", err)
	}

	if !installCalled {
		t.Error("Install was not called")
	}
	if len(capturedPkgs) != 2 {
		t.Errorf("Expected 2 pkgs, got %d", len(capturedPkgs))
	}
	if !capturedOpts.DownloadOnly {
		t.Error("Expected DownloadOnly=true")
	}
	if !strings.Contains(buf.String(), "Downloading packages...") {
		t.Errorf("Unexpected output: %s", buf.String())
	}
}
