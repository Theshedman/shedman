package resolver

import (
	"github.com/theshedman/shedman/pkg/shedman/pkgdb"
)

// Source priority order (highest to lowest)
var defaultSourcePriority = []string{
	pkgdb.SourceShedOS,
	pkgdb.SourceOfficial,
	pkgdb.SourceAUR,
}

// MultiSourceResolver queries multiple package databases with priority
type MultiSourceResolver struct {
	sources  map[string]pkgdb.PackageDB
	priority []string
}

// NewMultiSource creates a new MultiSourceResolver with default priority
func NewMultiSource() *MultiSourceResolver {
	return &MultiSourceResolver{
		sources:  make(map[string]pkgdb.PackageDB),
		priority: defaultSourcePriority,
	}
}

// AddSource adds a package database for a given source
func (m *MultiSourceResolver) AddSource(source string, db pkgdb.PackageDB) {
	m.sources[source] = db
}

// SetPriority sets custom source priority order
func (m *MultiSourceResolver) SetPriority(priority []string) {
	m.priority = priority
}

// FindPackage searches for a package across all sources with priority
// If forceSource is non-empty, only that source is queried
func (m *MultiSourceResolver) FindPackage(name string, forceSource string) (*pkgdb.PackageInfo, error) {
	// If source is forced, only check that source
	if forceSource != "" {
		if db, ok := m.sources[forceSource]; ok {
			return db.GetInfo(name)
		}
		return nil, nil
	}

	// Check sources in priority order
	for _, source := range m.priority {
		db, ok := m.sources[source]
		if !ok {
			continue
		}

		info, err := db.GetInfo(name)
		if err != nil {
			// Continue to next source on error (e.g. network failure)
			continue
		}
		if info != nil {
			return info, nil
		}
	}

	return nil, nil
}

// Search searches for packages across all sources
func (m *MultiSourceResolver) Search(query string) ([]pkgdb.PackageInfo, error) {
	var results []pkgdb.PackageInfo

	for _, source := range m.priority {
		db, ok := m.sources[source]
		if !ok {
			continue
		}

		sourceResults, err := db.Search(query)
		if err != nil {
			continue // Skip failed sources
		}

		results = append(results, sourceResults...)
	}

	return results, nil
}

// GetInfo implements pkgdb.PackageDB interface for compatibility
func (m *MultiSourceResolver) GetInfo(name string) (*pkgdb.PackageInfo, error) {
	return m.FindPackage(name, "")
}
