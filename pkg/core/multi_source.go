package core

import (
	"fmt"
	"strings"
)

var defaultSourcePriority = []string{
	SourceShedOS,
	SourceOfficial,
	SourceAUR,
}

// MultiSourceResolver queries multiple package databases with priority
type MultiSourceResolver struct {
	sources  map[string]PackageDB
	priority []string
}

// NewMultiSource creates a new MultiSourceResolver with default priority
func NewMultiSource() *MultiSourceResolver {
	return &MultiSourceResolver{
		sources:  make(map[string]PackageDB),
		priority: defaultSourcePriority,
	}
}

// AddSource adds a package database for a given source
func (m *MultiSourceResolver) AddSource(source string, db PackageDB) {
	m.sources[source] = db
}

// SetPriority sets custom source priority order
func (m *MultiSourceResolver) SetPriority(priority []string) {
	m.priority = priority
}

// FindPackage searches for a package across all sources with priority
// If forceSource is non-empty, only that source is queried
func (m *MultiSourceResolver) FindPackage(name string, forceSource string) (*PackageInfo, error) {
	// If source is forced, only check that source
	if forceSource != "" {
		if db, ok := m.sources[forceSource]; ok {
			return db.GetInfo(name)
		}
		return nil, fmt.Errorf("source %s not configured", forceSource)
	}

	// Check sources in priority order
	var errors []string
	for _, source := range m.priority {
		db, ok := m.sources[source]
		if !ok {
			continue
		}

		info, err := db.GetInfo(name)
		if err != nil {
			// Continue to next source on error (e.g. network failure)
			errors = append(errors, fmt.Sprintf("%s: %v", source, err))
			continue
		}
		if info != nil {
			return info, nil
		}
	}

	if len(errors) > 0 {
		return nil, fmt.Errorf("all sources failed: %s", strings.Join(errors, "; "))
	}
	return nil, nil
}

// Search searches for packages across all sources
func (m *MultiSourceResolver) Search(query string) ([]PackageInfo, error) {
	var results []PackageInfo
	var errors []string

	for _, source := range m.priority {
		db, ok := m.sources[source]
		if !ok {
			continue
		}

		sourceResults, err := db.Search(query)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", source, err))
			continue
		}

		results = append(results, sourceResults...)
	}

	if len(results) == 0 && len(errors) > 0 {
		return nil, fmt.Errorf("all sources failed: %s", strings.Join(errors, "; "))
	}
	if results == nil {
		return []PackageInfo{}, nil
	}
	return results, nil
}

// GetInfo implements PackageDB interface for compatibility
func (m *MultiSourceResolver) GetInfo(name string) (*PackageInfo, error) {
	return m.FindPackage(name, "")
}
