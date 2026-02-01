package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeConfigName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hypr", "shedos-configs-hypr"},
		{"shedos-configs-nvim", "shedos-configs-nvim"},
	}

	for _, tt := range tests {
		if got := normalizeConfigName(tt.input); got != tt.want {
			t.Errorf("normalizeConfigName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMapConfigTarget(t *testing.T) {
	home := "/home/tester"
	tests := []struct {
		path string
		want string
	}{
		{"/etc/skel/.config/nvim/init.lua", filepath.Join(home, ".config/nvim/init.lua")},
		{"/etc/xdg/app.conf", "/etc/xdg/app.conf"},
	}

	for _, tt := range tests {
		if got := mapConfigTarget(tt.path, home); got != tt.want {
			t.Errorf("mapConfigTarget(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestBackupsSelection(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "config")

	oldTS := "20240102120000.000"
	newTS := "20240102130000.000"

	oldBackup := target + "." + oldTS + ".bak"
	newBackup := target + "." + newTS + ".bak"

	if err := os.WriteFile(oldBackup, []byte("old"), 0600); err != nil {
		t.Fatalf("write old backup: %v", err)
	}
	if err := os.WriteFile(newBackup, []byte("new"), 0600); err != nil {
		t.Fatalf("write new backup: %v", err)
	}

	backups, err := listBackups(target)
	if err != nil {
		t.Fatalf("listBackups failed: %v", err)
	}
	if len(backups) != 2 {
		t.Fatalf("expected 2 backups, got %d", len(backups))
	}

	latest, err := selectBackup(target, "")
	if err != nil {
		t.Fatalf("selectBackup failed: %v", err)
	}
	if latest != newBackup {
		t.Errorf("latest backup = %s, want %s", latest, newBackup)
	}

	specific, err := selectBackup(target, oldTS)
	if err != nil {
		t.Fatalf("selectBackup specific failed: %v", err)
	}
	if specific != oldBackup {
		t.Errorf("specific backup = %s, want %s", specific, oldBackup)
	}
}

func TestRestoreBackup(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "config")
	backup := filepath.Join(tmpDir, "config.20240102120000.000.bak")

	if err := os.WriteFile(backup, []byte("backup-content"), 0600); err != nil {
		t.Fatalf("write backup: %v", err)
	}

	if err := restoreBackup(target, backup); err != nil {
		t.Fatalf("restoreBackup failed: %v", err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(data) != "backup-content" {
		t.Errorf("restored content = %q, want %q", string(data), "backup-content")
	}
}
