package config_test

import (
"os"
"path/filepath"
"testing"
"time"

"github.com/theshedman/shedman/pkg/shedman/config"
)

func TestConfig_Default(t *testing.T) {
	cfg := config.Default()

	if cfg == nil {
		t.Fatal("Default() should return non-nil config")
	}

	if len(cfg.ShedRepoMirrors) == 0 {
		t.Error("Default config should have ShedRepo mirrors")
	}

	if cfg.CacheMaxAge == 0 {
		t.Error("Default config should have CacheMaxAge set")
	}
}

func TestConfig_Load_FromFile(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "shedman-config-test")
	defer os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0755)

	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
shedrepo_mirrors:
  - https://custom.mirror.org
cache_max_age: 2h
`
	os.WriteFile(configPath, []byte(configContent), 0644)

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(cfg.ShedRepoMirrors) != 1 {
		t.Errorf("Expected 1 mirror, got %d", len(cfg.ShedRepoMirrors))
	}
	if cfg.ShedRepoMirrors[0] != "https://custom.mirror.org" {
		t.Errorf("Expected custom mirror, got %s", cfg.ShedRepoMirrors[0])
	}
	if cfg.CacheMaxAge != 2*time.Hour {
		t.Errorf("Expected 2h, got %v", cfg.CacheMaxAge)
	}
}

func TestConfig_Load_FileNotExists_ReturnsDefault(t *testing.T) {
	cfg, err := config.Load("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("Load should not error for missing file: %v", err)
	}

	// Should return defaults
	if len(cfg.ShedRepoMirrors) == 0 {
		t.Error("Should return default config when file not found")
	}
}
