package config

import (
	"os"

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
			AUR:    "https://aur.archlinux.org",
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
