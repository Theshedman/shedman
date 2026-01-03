package config

import "time"

// GeneralConfig holds general settings
type GeneralConfig struct {
	Color         bool   `toml:"color"`
	Confirm       bool   `toml:"confirm"`
	Editor        string `toml:"editor"`
	PromptTimeout int    `toml:"prompt_timeout"` // Timeout in seconds for interactive prompts (0 = no timeout)
}

// NetworkConfig holds network-related settings
type NetworkConfig struct {
	ParallelDownloads int    `toml:"parallel_downloads"`
	Timeout           int    `toml:"timeout"`
	Retry             int    `toml:"retry"`
	Proxy             string `toml:"proxy"`
	LimitRate         string `toml:"limit_rate"`
	DeltaUpdates      bool   `toml:"delta_updates"`
}

// CacheConfig holds cache settings
type CacheConfig struct {
	MaxAge    string `toml:"max_age"`
	CleanKeep int    `toml:"clean_keep"`
	AutoClean bool   `toml:"auto_clean"`
}

// GetMaxAge parses the MaxAge string into time.Duration
func (c *CacheConfig) GetMaxAge() time.Duration {
	d, err := time.ParseDuration(c.MaxAge)
	if err != nil {
		return 1 * time.Hour
	}
	return d
}

// MirrorConfig holds mirror URLs for different package sources
type MirrorConfig struct {
	ShedOS []string `toml:"shedos"`
	Arch   []string `toml:"arch"`
	AUR    string   `toml:"aur"`
	Apt    []string `toml:"apt"`
	Dnf    []string `toml:"dnf"`
	Zypper []string `toml:"zypper"`
}

// PackageConfig holds package management settings
type PackageConfig struct {
	IgnorePkg    []string `toml:"ignore_pkg"`
	IgnoreGroup  []string `toml:"ignore_group"`
	HoldPkg      []string `toml:"hold_pkg"`
	UpgradeFirst []string `toml:"upgrade_first"`
}

// BootConfig holds boot-related settings
type BootConfig struct {
	KeepKernels int `toml:"keep_kernels"`
}

// SnapshotConfig holds snapshot and backup settings
type SnapshotConfig struct {
	AutoBeforeUpdate  bool     `toml:"auto_before_update"`
	KeepLocal         int      `toml:"keep_local"`
	DefaultRemote     string   `toml:"default_remote"`
	Encrypt           bool     `toml:"encrypt"`
	ScheduleEnabled   bool     `toml:"schedule_enabled"`
	ScheduleFrequency string   `toml:"schedule_frequency"`
	ScheduleDays      []string `toml:"schedule_days"`
	ScheduleTime      string   `toml:"schedule_time"`
	ScheduleToRemote  bool     `toml:"schedule_to_remote"`
	RequireACPower    bool     `toml:"require_ac_power"`
	RequireWifi       bool     `toml:"require_wifi"`
	NotifyOnComplete  bool     `toml:"notify_on_complete"`
}

// NotificationConfig holds notification settings
type NotificationConfig struct {
	Enabled  bool   `toml:"enabled"`
	Interval string `toml:"interval"`
	Desktop  bool   `toml:"desktop"`
}

// GetInterval parses the Interval string into time.Duration
func (n *NotificationConfig) GetInterval() time.Duration {
	d, err := time.ParseDuration(n.Interval)
	if err != nil {
		return 6 * time.Hour
	}
	return d
}

// AURConfig holds AUR-specific settings
type AURConfig struct {
	Enabled         bool   `toml:"enabled"`
	BuildDir        string `toml:"build_dir"`
	CleanAfterBuild bool   `toml:"clean_after_build"`
	PGPFetch        bool   `toml:"pgp_fetch"`
}

// SecurityConfig holds security settings
type SecurityConfig struct {
	SigLevel string `toml:"sig_level"`
	GPGDir   string `toml:"gpg_dir"`
}

// HookConfig holds hook script paths
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

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Enabled bool   `toml:"enabled"`
	Level   string `toml:"level"`
	File    string `toml:"file"`
	JSON    bool   `toml:"json"`
}

// UIConfig holds UI settings
type UIConfig struct {
	ProgressBar bool   `toml:"progress_bar"`
	Spinner     bool   `toml:"spinner"`
	Pager       string `toml:"pager"`
}

// CloudConfig holds cloud storage settings
type CloudConfig struct {
	Provider string `toml:"provider"`
	Bucket   string `toml:"bucket"`
	Region   string `toml:"region"`
	Endpoint string `toml:"endpoint"`
}
