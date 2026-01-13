package cmd

import (
	"bytes"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

func TestRunExport(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	mock.ListExplicitPackagesFunc = func() ([]string, error) {
		return []string{"pkg1", "pkg2", "pkg3"}, nil
	}

	var buf bytes.Buffer
	err := RunExport(eng, &buf)
	if err != nil {
		t.Fatalf("RunExport failed: %v", err)
	}

	out := buf.String()
	expected := "pkg1\npkg2\npkg3\n"
	if out != expected {
		t.Errorf("Expected output:\n%q\nGot:\n%q", expected, out)
	}
}

func TestRunExport_Error(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	mock.ListExplicitPackagesFunc = func() ([]string, error) {
		return nil, core.ErrBackendNotFound
	}

	var buf bytes.Buffer
	err := RunExport(eng, &buf)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}
