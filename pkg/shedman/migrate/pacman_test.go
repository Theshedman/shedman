package migrate_test

import (
"os"
"path/filepath"
"testing"

"github.com/theshedman/shedman/pkg/shedman/migrate"
)

const samplePacmanConf = `
#
# /etc/pacman.conf
#
[options]
HoldPkg     = pacman glibc
Architecture = auto
ParallelDownloads = 5
SigLevel    = Required DatabaseOptional
IgnorePkg   = linux-headers

[core]
Server = https://geo.mirror.pkgbuild.com/$repo/os/$arch

[extra]
Include = /etc/pacman.d/mirrorlist

[multilib]
Include = /etc/pacman.d/mirrorlist
`

func TestParsePacmanConf_Options(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "shedman-pacman-test")
	defer os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0755)

	confPath := filepath.Join(tmpDir, "pacman.conf")
	os.WriteFile(confPath, []byte(samplePacmanConf), 0644)

	parsed, err := migrate.ParsePacmanConf(confPath)
	if err != nil {
		t.Fatalf("ParsePacmanConf failed: %v", err)
	}

	if parsed.ParallelDownloads != 5 {
		t.Errorf("Expected ParallelDownloads=5, got %d", parsed.ParallelDownloads)
	}

	if len(parsed.IgnorePkg) != 1 || parsed.IgnorePkg[0] != "linux-headers" {
		t.Errorf("Expected IgnorePkg=[linux-headers], got %v", parsed.IgnorePkg)
	}

	if len(parsed.HoldPkg) != 2 {
		t.Errorf("Expected HoldPkg=[pacman, glibc], got %v", parsed.HoldPkg)
	}

	if parsed.SigLevel != "Required DatabaseOptional" {
		t.Errorf("Expected SigLevel='Required DatabaseOptional', got %s", parsed.SigLevel)
	}
}

func TestParsePacmanConf_Mirrors(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "shedman-pacman-test")
	defer os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0755)

	confPath := filepath.Join(tmpDir, "pacman.conf")
	os.WriteFile(confPath, []byte(samplePacmanConf), 0644)

	parsed, err := migrate.ParsePacmanConf(confPath)
	if err != nil {
		t.Fatalf("ParsePacmanConf failed: %v", err)
	}

	if len(parsed.Mirrors) == 0 {
		t.Error("Expected at least one mirror")
	}

	found := false
	for _, m := range parsed.Mirrors {
		if m == "https://geo.mirror.pkgbuild.com/$repo/os/$arch" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected geo.mirror.pkgbuild.com in mirrors")
	}
}

func TestParsePacmanConf_FileNotExists(t *testing.T) {
	_, err := migrate.ParsePacmanConf("/nonexistent/pacman.conf")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}
