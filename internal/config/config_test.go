package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/internal/util"
)

func TestConfig_Default(t *testing.T) {
	cfg := config.Default()

	if cfg == nil {
		t.Fatal("Default() should return non-nil config")
	}

	if len(cfg.Mirrors.ShedOS) == 0 {
		t.Error("Default config should have ShedOS mirrors")
	}

	if cfg.Cache.GetMaxAge() == 0 {
		t.Error("Default config should have Cache.MaxAge set")
	}

	if cfg.Network.ParallelDownloads != 5 {
		t.Error("Default parallel downloads should be 5")
	}

	if cfg.General.Color != true {
		t.Error("Default color should be true")
	}
}

func TestConfig_Load_FromFile(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "shedman-config-test")
	defer os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, util.DirPermissions)

	configPath := filepath.Join(tmpDir, "config.toml")
	configContent := `
[mirrors]
shedos = ["https://custom.mirror.org"]

[cache]
max_age = "2h"

[network]
parallel_downloads = 10
`
	os.WriteFile(configPath, []byte(configContent), util.FilePermissions)

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(cfg.Mirrors.ShedOS) != 1 {
		t.Errorf("Expected 1 mirror, got %d", len(cfg.Mirrors.ShedOS))
	}
	if cfg.Mirrors.ShedOS[0] != "https://custom.mirror.org" {
		t.Errorf("Expected custom mirror, got %s", cfg.Mirrors.ShedOS[0])
	}
	if cfg.Cache.GetMaxAge() != 2*time.Hour {
		t.Errorf("Expected 2h, got %v", cfg.Cache.GetMaxAge())
	}
	if cfg.Network.ParallelDownloads != 10 {
		t.Errorf("Expected 10 parallel downloads, got %d", cfg.Network.ParallelDownloads)
	}
}

func TestConfig_Load_SnapshotKeys(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "shedman-snapshot-test")
	defer os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, util.DirPermissions)

	configPath := filepath.Join(tmpDir, "config.toml")
	configContent := `
[snapshot]
scheduled = true
schedule = "daily"
keep_scheduled = 5
auto_push = true
auto_push_remote = "r2"
`
	os.WriteFile(configPath, []byte(configContent), util.FilePermissions)

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if !cfg.Snapshot.Scheduled {
		t.Error("Expected scheduled=true")
	}
	if cfg.Snapshot.Schedule != "daily" {
		t.Errorf("Expected schedule=daily, got %s", cfg.Snapshot.Schedule)
	}
	if cfg.Snapshot.KeepScheduled != 5 {
		t.Errorf("Expected keep_scheduled=5, got %d", cfg.Snapshot.KeepScheduled)
	}
	if !cfg.Snapshot.AutoPush {
		t.Error("Expected auto_push=true")
	}
	if cfg.Snapshot.AutoPushRemote != "r2" {
		t.Errorf("Expected auto_push_remote=r2, got %s", cfg.Snapshot.AutoPushRemote)
	}
}

func TestConfig_Load_FileNotExists_ReturnsDefault(t *testing.T) {
	cfg, err := config.Load("/nonexistent/path/config.toml")
	if err != nil {
		t.Fatalf("Load should not error for missing file: %v", err)
	}

	if len(cfg.Mirrors.ShedOS) == 0 {
		t.Error("Should return default config when file not found")
	}
}

func TestConfig_AllSections(t *testing.T) {
	cfg := config.Default()

	// Test all 14 sections exist
	if cfg.General.Color != true {
		t.Error("General section missing")
	}
	if cfg.Network.Timeout != 30 {
		t.Error("Network section missing")
	}
	if cfg.Cache.CleanKeep != 3 {
		t.Error("Cache section missing")
	}
	if cfg.Boot.KeepKernels != 3 {
		t.Error("Boot section missing")
	}
	if cfg.Snapshot.KeepLocal != 10 {
		t.Error("Snapshot section missing")
	}
	if cfg.Snapshot.Schedule != "weekly" {
		t.Error("Snapshot default schedule incorrect")
	}
	if cfg.Notifications.Enabled != true {
		t.Error("Notifications section missing")
	}
	if cfg.AUR.Enabled != true {
		t.Error("AUR section missing")
	}
	if cfg.Security.SigLevel != "Required" {
		t.Error("Security section missing")
	}
	if cfg.Logging.Level != "info" {
		t.Error("Logging section missing")
	}
	if cfg.UI.ProgressBar != true {
		t.Error("UI section missing")
	}
}
