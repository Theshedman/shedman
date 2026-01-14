package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"github.com/theshedman/shedman/internal/config"
)

// LoadConfigForEdit loads the config file for editing.
// Note: This logic assumes a single config file location logic similar to internal/config/loader.go
// but strict for read-modify-write.
func LoadConfigForEdit() (*config.Config, error) {
	// 1. Determine path (Default /etc/shedman/shedman.conf or ~/.config/shedman/shedman.conf)
	// For user-level tools, prefer user config if strictly managing user scope,
	// but shedman often runs system-wide.
	// We'll mimic config.LoadDefault order but return the loaded path + struct.

	// Simplify: Load Default and check config file path?
	// config.LoadDefault doesn't return the path used easily without refactor.

	// Hardcode for this Helper: Try user, then system.

	home, _ := os.UserHomeDir()
	userPath := filepath.Join(home, ".config", "shedman", "shedman.conf")

	if _, err := os.Stat(userPath); err == nil {
		return config.Load(userPath)
	}

	return nil, fmt.Errorf("no writable user config found at %s (system config editing not yet supported via CLI)", userPath)
}

// SaveConfig saves the config struct to the user config file
func SaveConfig(cfg *config.Config) error {
	home, _ := os.UserHomeDir()
	userPath := filepath.Join(home, ".config", "shedman", "shedman.conf")

	// Ensure dir exists
	if err := os.MkdirAll(filepath.Dir(userPath), 0755); err != nil {
		return err
	}

	f, err := os.Create(userPath)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := toml.NewEncoder(f)
	// enc.SetIndentTables(true) // If available in v2
	return enc.Encode(cfg)
}
