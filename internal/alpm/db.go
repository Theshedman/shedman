// Package pacman provides database query functionality for pacman.
package alpm

import (
	"strconv"
	"strings"

	"github.com/Jguer/go-alpm/v2"
)

// DB queries the pacman package database
type DB struct {
	executor CommandExecutor
}

// NewDB creates a new pacman database interface
func NewDB() *DB {
	return &DB{
		executor: &RealExecutor{},
	}
}

// NewDBWithExecutor creates a DB with a custom executor (for testing)
func NewDBWithExecutor(exec CommandExecutor) *DB {
	return &DB{executor: exec}
}

// Search searches for packages matching the query
func (d *DB) Search(query string) ([]PackageInfo, error) {
	output, err := d.executor.Output("pacman", "-Ss", query)
	if err != nil {
		return nil, err
	}

	return ParseSearchOutput(string(output)), nil
}

// GetInfo returns detailed info about a package
func (d *DB) GetInfo(name string) (*PackageInfo, error) {
	output, err := d.executor.Output("pacman", "-Si", name)
	if err != nil {
		return nil, err
	}

	if strings.Contains(string(output), "was not found") {
		return nil, nil
	}

	info := ParseInfo(string(output))
	info.Source = SourceOfficial
	return &info, nil
}

// IsInstalled checks if a package is installed locally
func (d *DB) IsInstalled(name string) bool {
	err := d.executor.Run("pacman", "-Q", name)
	return err == nil
}

// GetInstalledVersion returns the installed version of a package
func (d *DB) GetInstalledVersion(name string) string {
	output, err := d.executor.Output("pacman", "-Q", name)
	if err != nil {
		return ""
	}

	parts := strings.Fields(string(output))
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// ParseSearchOutput parses pacman -Ss output
func ParseSearchOutput(output string) []PackageInfo {
	var results []PackageInfo
	lines := strings.Split(output, "\n")

	for i := 0; i < len(lines)-1; i += 2 {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 2 {
			nameParts := strings.Split(parts[0], "/")
			name := nameParts[len(nameParts)-1]

			info := PackageInfo{
				Name:    name,
				Version: parts[1],
				Source:  SourceOfficial,
			}

			if i+1 < len(lines) {
				info.Description = strings.TrimSpace(lines[i+1])
			}

			results = append(results, info)
		}
	}

	return results
}

// ParseInfo parses pacman -Si output into PackageInfo
func ParseInfo(output string) PackageInfo {
	info := PackageInfo{}
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if idx := strings.Index(line, ":"); idx != -1 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])

			switch key {
			case "Name":
				info.Name = value
			case "Version":
				info.Version = value
			case "Description":
				info.Description = value
			case "Depends On":
				if value != "None" && value != "" {
					info.Depends = strings.Fields(value)
				}
			case "Optional Deps":
				if value != "None" && value != "" {
					parts := strings.Split(value, ":")
					if len(parts) > 0 {
						info.OptDepends = append(info.OptDepends, strings.TrimSpace(parts[0]))
					}
				}
			case "Provides":
				if value != "None" && value != "" {
					info.Provides = strings.Fields(value)
				}
			case "Conflicts":
				if value != "None" && value != "" {
					info.Conflicts = strings.Fields(value)
				}
			case "Download Size":
				info.Size = ParseSize(value)
			case "Installed Size":
				info.InstalledSize = ParseSize(value)
			}
		}
	}

	return info
}

// ParseSize converts size strings like "10.5 MiB" to bytes
func ParseSize(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	parts := strings.Fields(s)
	if len(parts) < 2 {
		return 0
	}

	value, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0
	}

	var multiplier float64
	switch parts[1] {
	case "B":
		multiplier = 1
	case "KiB":
		multiplier = 1024
	case "MiB":
		multiplier = 1024 * 1024
	case "GiB":
		multiplier = 1024 * 1024 * 1024
	case "TiB":
		multiplier = 1024 * 1024 * 1024 * 1024
	default:
		return 0
	}

	return int64(value * multiplier)
}

// alpmAvailable caches the result of libalpm availability check
var (
	alpmChecked   bool
	alpmAvailable bool
)

// IsAlpmAvailable checks if libalpm can be initialized
func IsAlpmAvailable() bool {
	if alpmChecked {
		return alpmAvailable
	}
	// Try to initialize ALPM
	h, err := alpm.Initialize("/", "/var/lib/pacman")
	if err == nil {
		_ = h.Release()

		alpmAvailable = true
	} else {
		alpmAvailable = false
	}
	alpmChecked = true
	return alpmAvailable
}
