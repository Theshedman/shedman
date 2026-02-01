package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theshedman/shedman/internal/config"
)

func TestRunMigrate_DryRun(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	confPath := filepath.Join(tmpHome, "pacman.conf")
	confContent := `
[options]
IgnorePkg = linux
IgnoreGroup = base
HoldPkg = pacman
GPGDir = /etc/pacman.d/gnupg
SigLevel = Required DatabaseOptional
LogFile = /var/log/pacman.log
ParallelDownloads = 8

[core]
Server = https://mirror.example.com/$repo/os/$arch
`
	if err := os.WriteFile(confPath, []byte(confContent), 0644); err != nil {
		t.Fatalf("write pacman.conf: %v", err)
	}

	cfg := config.Default()
	var buf bytes.Buffer

	if err := RunMigrate(&buf, cfg, confPath, true); err != nil {
		t.Fatalf("RunMigrate dry-run failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Importing pacman config") {
		t.Errorf("expected import output, got: %s", output)
	}
	if strings.Contains(output, "Migration complete") {
		t.Errorf("did not expect migration completion in dry-run")
	}

	if _, err := os.Stat(config.DefaultConfigPath()); err == nil {
		t.Errorf("config file should not be written on dry-run")
	}
}

func TestRunMigrate_WritesConfig(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	confPath := filepath.Join(tmpHome, "pacman.conf")
	confContent := `
[options]
IgnorePkg = linux linux-headers
IgnoreGroup = base
HoldPkg = pacman
GPGDir = /etc/pacman.d/gnupg
SigLevel = Required DatabaseOptional
LogFile = /var/log/pacman.log
ParallelDownloads = 6

[core]
Server = https://mirror.example.com/$repo/os/$arch
`
	if err := os.WriteFile(confPath, []byte(confContent), 0644); err != nil {
		t.Fatalf("write pacman.conf: %v", err)
	}

	cfg := config.Default()
	var buf bytes.Buffer

	if err := RunMigrate(&buf, cfg, confPath, false); err != nil {
		t.Fatalf("RunMigrate failed: %v", err)
	}

	loaded, err := config.LoadDefault()
	if err != nil {
		t.Fatalf("LoadDefault failed: %v", err)
	}

	if len(loaded.Packages.IgnorePkg) != 2 {
		t.Errorf("expected ignore packages to be imported")
	}
	if loaded.Network.ParallelDownloads != 6 {
		t.Errorf("parallel downloads = %d, want 6", loaded.Network.ParallelDownloads)
	}
	if loaded.Logging.File != "/var/log/pacman.log" {
		t.Errorf("logging file = %s, want /var/log/pacman.log", loaded.Logging.File)
	}
}
