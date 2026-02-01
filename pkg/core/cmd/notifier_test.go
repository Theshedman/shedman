package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteUpdateCount(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "update-count")

	if err := writeUpdateCount(path, 5); err != nil {
		t.Fatalf("writeUpdateCount failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read update-count failed: %v", err)
	}
	if strings.TrimSpace(string(data)) != "5" {
		t.Errorf("update-count content = %q, want %q", string(data), "5")
	}
}

func TestMotdScriptContent(t *testing.T) {
	script := motdScriptContent()
	if !strings.Contains(script, "update-count") {
		t.Error("expected motd script to reference update-count cache")
	}
	if !strings.Contains(script, "shedman update") {
		t.Error("expected motd script to reference shedman update")
	}
}
