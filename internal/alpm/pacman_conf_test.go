package alpm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/theshedman/shedman/internal/util"
)

func TestParsePacmanConf(t *testing.T) {
	// Create a temporary pacman.conf
	tmpDir, err := os.MkdirTemp("", "pacman-conf-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	confPath := filepath.Join(tmpDir, "pacman.conf")
	confContent := `
# Test pacman.conf

[options]
RootDir     = /
DBPath      = /var/lib/pacman/
CacheDir    = /var/cache/pacman/pkg/
LogFile     = /var/log/pacman.log
GPGDir      = /etc/pacman.d/gnupg/
HoldPkg     = pacman glibc
IgnorePkg   = linux linux-headers
Architecture = auto
SigLevel    = Required DatabaseOptional

[core]
Server = https://mirror.example.com/$repo/os/$arch

[extra]
Server = https://mirror.example.com/$repo/os/$arch

[multilib]
Server = https://mirror.example.com/$repo/os/$arch
`
	if err := os.WriteFile(confPath, []byte(confContent), util.FilePermissions); err != nil {
		t.Fatal(err)
	}

	conf, err := ParsePacmanConf(confPath)
	if err != nil {
		t.Fatalf("ParsePacmanConf failed: %v", err)
	}

	// Test options
	if conf.RootDir != "/" {
		t.Errorf("RootDir = %s, want /", conf.RootDir)
	}
	if conf.DBPath != "/var/lib/pacman/" {
		t.Errorf("DBPath = %s, want /var/lib/pacman/", conf.DBPath)
	}
	if len(conf.HoldPkg) != 2 {
		t.Errorf("HoldPkg count = %d, want 2", len(conf.HoldPkg))
	}
	if len(conf.IgnorePkg) != 2 {
		t.Errorf("IgnorePkg count = %d, want 2", len(conf.IgnorePkg))
	}

	// Test repositories
	if len(conf.Repositories) != 3 {
		t.Fatalf("Repositories count = %d, want 3", len(conf.Repositories))
	}
	if conf.Repositories[0].Name != "core" {
		t.Errorf("First repo = %s, want core", conf.Repositories[0].Name)
	}
	if len(conf.Repositories[0].Servers) != 1 {
		t.Errorf("Core servers count = %d, want 1", len(conf.Repositories[0].Servers))
	}
}

func TestDefaultPacmanConf(t *testing.T) {
	conf := DefaultPacmanConf()

	if conf.RootDir != "/" {
		t.Errorf("RootDir = %s, want /", conf.RootDir)
	}
	if conf.DBPath != "/var/lib/pacman" {
		t.Errorf("DBPath = %s, want /var/lib/pacman", conf.DBPath)
	}
	if len(conf.Repositories) != 3 {
		t.Errorf("Repositories count = %d, want 3", len(conf.Repositories))
	}
}

func TestExpandVariables(t *testing.T) {
	conf := &PacmanConf{
		Architecture: "x86_64",
	}

	tests := []struct {
		url      string
		repoName string
		expected string
	}{
		{
			url:      "https://mirror.example.com/$repo/os/$arch",
			repoName: "core",
			expected: "https://mirror.example.com/core/os/x86_64",
		},
		{
			url:      "https://mirror.example.com/archlinux/$repo/os/$arch",
			repoName: "extra",
			expected: "https://mirror.example.com/archlinux/extra/os/x86_64",
		},
	}

	for _, tc := range tests {
		result := conf.ExpandVariables(tc.url, tc.repoName)
		if result != tc.expected {
			t.Errorf("ExpandVariables(%s, %s) = %s, want %s", tc.url, tc.repoName, result, tc.expected)
		}
	}
}

func TestGetMirrorsForRepo(t *testing.T) {
	conf := &PacmanConf{
		Architecture: "x86_64",
		Repositories: []RepoConfig{
			{
				Name:    "core",
				Servers: []string{"https://mirror1.com/$repo/os/$arch", "https://mirror2.com/$repo/os/$arch"},
			},
			{
				Name:    "extra",
				Servers: []string{"https://mirror1.com/$repo/os/$arch"},
			},
		},
	}

	coreMirrors := conf.GetMirrorsForRepo("core")
	if len(coreMirrors) != 2 {
		t.Errorf("Core mirrors count = %d, want 2", len(coreMirrors))
	}
	if coreMirrors[0] != "https://mirror1.com/core/os/x86_64" {
		t.Errorf("Core mirror[0] = %s, want https://mirror1.com/core/os/x86_64", coreMirrors[0])
	}

	extraMirrors := conf.GetMirrorsForRepo("extra")
	if len(extraMirrors) != 1 {
		t.Errorf("Extra mirrors count = %d, want 1", len(extraMirrors))
	}

	nonexistent := conf.GetMirrorsForRepo("nonexistent")
	if nonexistent != nil {
		t.Errorf("Nonexistent repo should return nil")
	}
}

func TestParseSpaceSeparated(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{"one", []string{"one"}},
		{"one two three", []string{"one", "two", "three"}},
		{"  spaced   values  ", []string{"spaced", "values"}},
	}

	for _, tc := range tests {
		result := parseSpaceSeparated(tc.input)
		if len(result) != len(tc.expected) {
			t.Errorf("parseSpaceSeparated(%q) = %v, want %v", tc.input, result, tc.expected)
			continue
		}
		for i := range result {
			if result[i] != tc.expected[i] {
				t.Errorf("parseSpaceSeparated(%q)[%d] = %s, want %s", tc.input, i, result[i], tc.expected[i])
			}
		}
	}
}
