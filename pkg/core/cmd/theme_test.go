package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

func TestPreviewTheme(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	mock.InfoFunc = func(name string) (*core.PackageInfo, error) {
		return &core.PackageInfo{
			Name:        name,
			Version:     "1.0.0",
			Description: "Test theme",
		}, nil
	}

	var buf bytes.Buffer
	if err := previewTheme(&buf, eng, "catppuccin"); err != nil {
		t.Fatalf("previewTheme failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "catppuccin") || !strings.Contains(out, "1.0.0") {
		t.Errorf("unexpected preview output: %s", out)
	}
}
