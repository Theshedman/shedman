package resolver

import (
	"fmt"

	"github.com/theshedman/shedman/pkg/shedman/pkgdb"
)

// Conflict represents a conflict between two packages
type Conflict struct {
	Package1 string
	Package2 string
	Reason   string
}

// String returns a human-readable description of the conflict
func (c Conflict) String() string {
	return fmt.Sprintf("%s conflicts with %s: %s", c.Package1, c.Package2, c.Reason)
}

// ConflictDetector detects conflicts between packages
type ConflictDetector struct {
	db pkgdb.PackageDB
}

// NewConflictDetector creates a new ConflictDetector
func NewConflictDetector(db pkgdb.PackageDB) *ConflictDetector {
	return &ConflictDetector{db: db}
}

// Detect checks for conflicts among the given packages
func (cd *ConflictDetector) Detect(packages []string) []Conflict {
	var conflicts []Conflict
	pkgInfos := make(map[string]*pkgdb.PackageInfo)
	provides := make(map[string]string) // provided name -> package that provides it

	// Load package info
	for _, name := range packages {
		info, _ := cd.db.GetInfo(name)
		if info != nil {
			pkgInfos[name] = info
		}
	}

	// Check for conflicts and provides
	for _, name := range packages {
		info := pkgInfos[name]
		if info == nil {
			continue
		}

		// Check explicit conflicts
		for _, conflictName := range info.Conflicts {
			if _, exists := pkgInfos[conflictName]; exists {
				conflicts = append(conflicts, Conflict{
					Package1: name,
					Package2: conflictName,
					Reason:   "explicit conflict",
				})
			}
		}

		// Track provides
		for _, prov := range info.Provides {
			if existingPkg, exists := provides[prov]; exists {
				conflicts = append(conflicts, Conflict{
					Package1: existingPkg,
					Package2: name,
					Reason:   fmt.Sprintf("both provide %s", prov),
				})
			} else {
				provides[prov] = name
			}
		}
	}

	return conflicts
}