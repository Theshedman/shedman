package pkgdb

import (
	"errors"
	"os/exec"
	"strconv"
	"strings"
)

// ErrPacmanNotFound is returned when pacman is not available
var ErrPacmanNotFound = errors.New("pacman is required but not found in PATH")

// CommandExecutor runs commands and returns output
type CommandExecutor func(cmd []string) (string, error)

// DefaultCommandExecutor runs commands using os/exec
func DefaultCommandExecutor(cmd []string) (string, error) {
	if len(cmd) == 0 {
		return "", nil
	}
	c := exec.Command(cmd[0], cmd[1:]...)
	output, err := c.Output()
	return string(output), err
}

// PacmanDB queries the pacman package database
type PacmanDB struct {
	executor CommandExecutor
}

// NewPacmanDB creates a new PacmanDB
func NewPacmanDB() *PacmanDB {
	return &PacmanDB{
		executor: DefaultCommandExecutor,
	}
}

// SetExecutor sets a custom command executor (for testing)
func (p *PacmanDB) SetExecutor(exec CommandExecutor) {
	p.executor = exec
}

// IsPacmanAvailable checks if pacman command exists
func (p *PacmanDB) IsPacmanAvailable() bool {
	_, err := exec.LookPath("pacman")
	return err == nil
}

// Search searches for packages matching the query
func (p *PacmanDB) Search(query string) ([]PackageInfo, error) {
	cmd := []string{"pacman", "-Ss", query}
	output, err := p.executor(cmd)
	if err != nil {
		return nil, err
	}

	return parseSearchOutput(output), nil
}

// GetInfo returns detailed info about a package
func (p *PacmanDB) GetInfo(name string) (*PackageInfo, error) {
	cmd := []string{"pacman", "-Si", name}
	output, err := p.executor(cmd)
	if err != nil {
		return nil, err
	}

	if strings.Contains(output, "was not found") {
		return nil, nil
	}

	info := ParsePacmanInfo(output)
	info.Source = SourceOfficial
	return &info, nil
}

// IsInstalled checks if a package is installed locally
func (p *PacmanDB) IsInstalled(name string) bool {
	cmd := []string{"pacman", "-Q", name}
	output, err := p.executor(cmd)
	if err != nil {
		return false
	}
	return strings.TrimSpace(output) != ""
}

// parseSearchOutput parses pacman -Ss output
func parseSearchOutput(output string) []PackageInfo {
	var results []PackageInfo
	lines := strings.Split(output, "\n")

	for i := 0; i < len(lines)-1; i += 2 {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Format: repo/name version
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			nameParts := strings.Split(parts[0], "/")
			name := nameParts[len(nameParts)-1]

			info := PackageInfo{
				Name:    name,
				Version: parts[1],
				Source:  SourceOfficial,
			}

			// Description is on next line
			if i+1 < len(lines) {
				info.Description = strings.TrimSpace(lines[i+1])
			}

			results = append(results, info)
		}
	}

	return results
}

// ParsePacmanInfo parses pacman -Si output into PackageInfo
func ParsePacmanInfo(output string) PackageInfo {
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
					// Parse "pkg: description" format
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

// ParseSize converts size strings like "10.5 MiB" to bytes (exported for testing)
func ParseSize(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	parts := strings.Fields(s)
	if len(parts) < 2 {
		return 0
	}

	valueStr := parts[0]
	unit := parts[1]

	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return 0
	}

	// Convert based on unit
	var multiplier float64
	switch unit {
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
