package cmd

import (
	"testing"

	appconfig "github.com/theshedman/shedman/internal/config"
)

func TestSetConfigValue(t *testing.T) {
	cfg := appconfig.Default()

	if err := setConfigValue(cfg, "boot.keep-kernels", "5"); err != nil {
		t.Fatalf("setConfigValue boot.keep-kernels failed: %v", err)
	}
	if cfg.Boot.KeepKernels != 5 {
		t.Errorf("KeepKernels = %d, want 5", cfg.Boot.KeepKernels)
	}

	if err := setConfigValue(cfg, "notifications.enabled", "false"); err != nil {
		t.Fatalf("setConfigValue notifications.enabled failed: %v", err)
	}
	if cfg.Notifications.Enabled {
		t.Errorf("Notifications.Enabled = true, want false")
	}

	if err := setConfigValue(cfg, "packages.ignore-pkg", "vim, git"); err != nil {
		t.Fatalf("setConfigValue packages.ignore-pkg failed: %v", err)
	}
	if len(cfg.Packages.IgnorePkg) != 2 || cfg.Packages.IgnorePkg[0] != "vim" || cfg.Packages.IgnorePkg[1] != "git" {
		t.Errorf("IgnorePkg = %v, want [vim git]", cfg.Packages.IgnorePkg)
	}
}

func TestGetConfigValue(t *testing.T) {
	cfg := appconfig.Default()
	cfg.Boot.KeepKernels = 4
	cfg.Packages.IgnorePkg = []string{"vim", "git"}

	val, err := getConfigValue(cfg, "boot.keep-kernels")
	if err != nil {
		t.Fatalf("getConfigValue boot.keep-kernels failed: %v", err)
	}
	if val != "4" {
		t.Errorf("boot.keep-kernels = %q, want \"4\"", val)
	}

	val, err = getConfigValue(cfg, "packages.ignore-pkg")
	if err != nil {
		t.Fatalf("getConfigValue packages.ignore-pkg failed: %v", err)
	}
	if val != "vim,git" {
		t.Errorf("packages.ignore-pkg = %q, want \"vim,git\"", val)
	}
}
