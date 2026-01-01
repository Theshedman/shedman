package config

import (
"os"
"time"

"github.com/pelletier/go-toml/v2"
)

// Config holds all shedman configuration
type Config struct {
	General       GeneralConfig       `toml:"general"`
	Network       NetworkConfig       `toml:"network"`
	Cache         CacheConfig         `toml:"cache"`
	Mirrors       MirrorConfig        `toml:"mirrors"`
	Packages      PackageConfig       `toml:"packages"`
	Boot          BootConfig          `toml:"boot"`
	Snapshot      SnapshotConfig      `toml:"snapshot"`
	Notifications NotificationConfig  `toml:"notifications"`
	AUR           AURConfig           `toml:"aur"`
	Security      SecurityConfig      `toml:"security"`
	Hooks         HookConfig          `toml:"hooks"`
	Logging       LoggingConfig       `toml:"logging"`
	UI            UIConfig            `toml:"ui"`
	Cloud         CloudConfig         `toml:"cloud"`
}

type GeneralConfig struct {
	Color   bool   `toml:"color"`
	Confirm bool   `toml:"confirm"`
	Editor  string `toml:"editor"`
}

type NetworkConfig struct {
	ParallelDownloads int    `toml:"parallel_downloads"`
	Timeout           int    `toml:"timeout"`
	Retry             int    `toml:"retry"`
	Proxy             string `toml:"proxy"`
	LimitRate         string `toml:"limit_rate"`
	DeltaUpdates      bool   `toml:"delta_updates"`
}

type CacheConfig struct {
	MaxAge    string `toml:"max_age"` // Duration as string, e.g., "1h"
	CleanKeep int    `toml:"clean_keep"`
	AutoClean bool   `toml:"auto_clean"`
}

// GetMaxAge parses the MaxAge string into time.Duration
func (c *CacheConfig) GetMaxAge() time.Duration {
	d, err := time.ParseDuration(c.MaxAge)
	if err != nil {
		return 1 * time.Hour // default
	}
	return d
}

type MirrorConfig struct {
	ShedOS []string `toml:"shedos"`
	Arch   []string `toml:"arch"`
	Apt    []string `toml:"apt"`
	Dnf    []string `toml:"dnf"`
	Zypper []string `toml:"zypper"`
}

type PackageConfig struct {
	IgnorePkg    []string `toml:"ignore_pkg"`
	IgnoreGroup  []string `toml:"ignore_group"`
	HoldPkg      []string `toml:"hold_pkg"`
	UpgradeFirst []string `toml:"upgrade_first"`
}

type BootConfig struct {
	KeepKernels int `toml:"keep_kernels"`
}

type SnapshotConfig struct {
	AutoBeforeUpdate bool   `toml:"auto_before_update"`
	KeepLocal        int    `toml:"keep_local"`
	DefaultRemote    string `toml:"default_remote"`
	Encrypt          bool   `toml:"encrypt"`
}

type NotificationConfig struct {
	Enabled  bool   `toml:"enabled"`
	Interval string `toml:"interval"` // Duration as string
	Desktop  bool   `toml:"desktop"`
}

// GetInterval parses the Interval string into time.Duration
func (n *NotificationConfig) GetInterval() time.Duration {
	d, err := time.ParseDuration(n.Interval)
	if err != nil {
		return 6 * time.Hour // default
	}
	return d
}

type AURConfig struct {
	Enabled         bool   `toml:"enabled"`
	BuildDir        string `toml:"build_dir"`
	CleanAfterBuild bool   `toml:"clean_after_build"`
	PGPFetch        bool   `toml:"pgp_fetch"`
}

type SecurityConfig struct {
	SigLevel string `toml:"sig_level"`
	GPGDir   string `toml:"gpg_dir"`
}

type HookConfig struct {
	PreInstall  string `toml:"pre_install"`
	PostInstall string `toml:"post_install"`
	PreRemove   string `toml:"pre_remove"`
	PostRemove  string `toml:"post_remove"`
	PreUpgrade  string `toml:"pre_upgrade"`
	PostUpgrade string `toml:"post_upgrade"`
	PreSync     string `toml:"pre_sync"`
	PostSync    string `toml:"post_sync"`
}

type LoggingConfig struct {
	Enabled bool   `toml:"enabled"`
	Level   string `toml:"level"`
	File    string `toml:"file"`
	JSON    bool   `toml:"json"`
}

type UIConfig struct {
	ProgressBar bool   `toml:"progress_bar"`
	Spinner     bool   `toml:"spinner"`
	Pager       string `toml:"pager"`
}

type CloudConfig struct {
	Provider string `toml:"provider"`
	Bucket   string `toml:"bucket"`
	Region   string `toml:"region"`
	Endpoint string `toml:"endpoint"`
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
		},
		Boot: BootConfig{
			KeepKernels: 3,
		},
		Snapshot: SnapshotConfig{
			AutoBeforeUpdate: true,
			KeepLocal:        10,
			DefaultRemote:    "gdrive",
			Encrypt:          true,
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
