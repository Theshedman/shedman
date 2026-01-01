package config

import (
"os"
"time"

"gopkg.in/yaml.v3"
)

// Config holds all shedman configuration
type Config struct {
	ShedRepoMirrors []string      `yaml:"shedrepo_mirrors"`
	CacheMaxAge     time.Duration `yaml:"cache_max_age"`
}

// Default returns the default configuration
func Default() *Config {
	return &Config{
		ShedRepoMirrors: []string{
			"https://repo.shedos.org",
			"https://mirror1.shedos.org",
			"https://mirror2.shedos.org",
		},
		CacheMaxAge: 1 * time.Hour,
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

cfg := Default() // Start with defaults
if err := yaml.Unmarshal(data, cfg); err != nil {
return nil, err
}

return cfg, nil
}
