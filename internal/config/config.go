package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// Config holds all shedman configuration
type Config struct {
	General       GeneralConfig      `toml:"general"`
	Network       NetworkConfig      `toml:"network"`
	Cache         CacheConfig        `toml:"cache"`
	Mirrors       MirrorConfig       `toml:"mirrors"`
	Packages      PackageConfig      `toml:"packages"`
	Boot          BootConfig         `toml:"boot"`
	Snapshot      SnapshotConfig     `toml:"snapshot"`
	Notifications NotificationConfig `toml:"notifications"`
	AUR           AURConfig          `toml:"aur"`
	Security      SecurityConfig     `toml:"security"`
	Hooks         HookConfig         `toml:"hooks"`
	Logging       LoggingConfig      `toml:"logging"`
	UI            UIConfig           `toml:"ui"`
	Cloud         CloudConfig        `toml:"cloud"`
	Backend       BackendConfig      `toml:"backend"`
}

// Default returns the default configuration
func Default() *Config {
	return &Config{
		General: GeneralConfig{
			Color:   true,
			Confirm: true,
		},
		Network: NetworkConfig{
			ParallelDownloads: 5,
			Timeout:           30,
			Retry:             3,
		},
		Cache: CacheConfig{
			MaxAge:    "1h",
			CleanKeep: 3,
		},
		Mirrors: MirrorConfig{
			ShedOS: []string{"https://repo.shedos.org"},
			Arch:   []string{"https://geo.mirror.pkgbuild.com/$repo/os/$arch"},
			AUR:    "https://aur.archlinux.org/rpc/",
		},
		Boot: BootConfig{
			KeepKernels: 3,
		},
		Snapshot: SnapshotConfig{
			AutoBeforeUpdate:  true,
			KeepLocal:         10,
			DefaultRemote:     "gdrive",
			Encrypt:           true,
			ScheduleEnabled:   false,
			ScheduleFrequency: "weekly",
			ScheduleDays:      []string{"friday"},
			ScheduleTime:      "15:00",
			ScheduleToRemote:  true,
			RequireACPower:    false,
			RequireWifi:       false,
			NotifyOnComplete:  true,
		},
		Notifications: NotificationConfig{
			Enabled:  true,
			Interval: "6h",
			Desktop:  true,
		},
		AUR: AURConfig{
			Enabled:         true,
			CleanAfterBuild: true,
			PGPFetch:        true,
		},
		Security: SecurityConfig{
			SigLevel: "Required",
		},
		Logging: LoggingConfig{
			Enabled: true,
			Level:   "info",
		},
		UI: UIConfig{
			ProgressBar: true,
			Spinner:     true,
		},
		Backend: BackendConfig{
			AutoDetect: true, // Auto-detect package manager by default
		},
	}
}

// Load reads configuration from a file, returning defaults if file doesn't exist
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return nil, err
	}

	cfg := Default()
	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save writes the configuration to the specified path
func Save(path string, cfg *Config) error {
	data, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// DefaultConfigPath returns the default config file path (~/.config/shedman/config.toml)
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/etc/shedman/config.toml" // Fallback to system config
	}
	return filepath.Join(home, ".config", "shedman", "config.toml")
}

// LoadDefault loads config from the default path (~/.config/shedman/config.toml)
// If the file doesn't exist, it CREATES it with default values.
func LoadDefault() (*Config, error) {
	path := DefaultConfigPath()

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Create default config
		cfg := Default()

		// Save it to disk
		if err := Save(path, cfg); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}

		return cfg, nil
	}

	return Load(path)
}
