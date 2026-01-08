// Package pkgdb provides package database querying functionality.
package pkgdb

import (
	"errors"
	"strconv"
	"strings"
)

// ErrPacmanNotFound is returned when pacman is not available
var ErrPacmanNotFound = errors.New("pacman is required but not found in PATH")

// PackageSearcher can search for packages
type PackageSearcher interface {
	Search(query string) ([]PackageInfo, error)
	Info(name string) (*PackageInfo, error)
	IsInstalled(name string) bool
	IsAvailable() bool
}

// PacmanDB queries the pacman package database.
// It wraps a PackageSearcher backend for PackageDB interface compatibility.
type PacmanDB struct {
	backend PackageSearcher
}

// NewPacmanDBWithBackend creates a PacmanDB with a specific backend
// The backend must implement PackageSearcher (e.g., backend/pacman.Backend)
func NewPacmanDBWithBackend(b PackageSearcher) *PacmanDB {
	return &PacmanDB{backend: b}
}

// NewPacmanDB creates a new PacmanDB.
// Note: This returns a PacmanDB with nil backend. Use NewPacmanDBWithBackend
// and pass a pacman.Backend for full functionality.
// This exists for backward compatibility but callers should update to use
// NewPacmanDBWithBackend with an appropriate backend.
func NewPacmanDB() *PacmanDB {
	// Return with nil backend - callers should use NewPacmanDBWithBackend
	return &PacmanDB{backend: nil}
}

// SetBackend sets the backend (for lazy initialization)
func (p *PacmanDB) SetBackend(b PackageSearcher) {
	p.backend = b
}

// IsAlpmAvailable checks if the backend is available
func (p *PacmanDB) IsAlpmAvailable() bool {
	return p.backend != nil && p.backend.IsAvailable()
}

// Search searches for packages matching the query
func (p *PacmanDB) Search(query string) ([]PackageInfo, error) {
	if p.backend == nil {
		return nil, ErrPacmanNotFound
	}

	return p.backend.Search(query)
}

// GetInfo returns detailed info about a package
func (p *PacmanDB) GetInfo(name string) (*PackageInfo, error) {
	if p.backend == nil {
		return nil, ErrPacmanNotFound
	}

	return p.backend.Info(name)
}

// IsInstalled checks if a package is installed locally
func (p *PacmanDB) IsInstalled(name string) bool {
	if p.backend == nil {
		return false
	}
	return p.backend.IsInstalled(name)
}

// ParsePacmanInfo parses pacman -Si output into PackageInfo
// Kept for backward compatibility and testing
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

	valueStr := parts[0]
	unit := parts[1]

	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return 0
	}

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
