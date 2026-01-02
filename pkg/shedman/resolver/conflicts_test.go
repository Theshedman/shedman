package resolver_test

import (
	"testing"

	"github.com/theshedman/shedman/pkg/shedman/pkgdb"
	"github.com/theshedman/shedman/pkg/shedman/resolver"
)

func TestConflictDetector_NoConflicts(t *testing.T) {
	db := &conflictTestDB{
		packages: map[string]pkgdb.PackageInfo{
			"neovim": {Name: "neovim"},
			"git":    {Name: "git"},
		},
	}

	detector := resolver.NewConflictDetector(db)
	conflicts := detector.Detect([]string{"neovim", "git"})

	if len(conflicts) != 0 {
		t.Errorf("Expected no conflicts, got %v", conflicts)
	}
}

func TestConflictDetector_PackageConflict(t *testing.T) {
	db := &conflictTestDB{
		packages: map[string]pkgdb.PackageInfo{
			"vim":    {Name: "vim", Conflicts: []string{"neovim"}},
			"neovim": {Name: "neovim", Conflicts: []string{"vim"}},
		},
	}

	detector := resolver.NewConflictDetector(db)
	conflicts := detector.Detect([]string{"vim", "neovim"})

	// Both packages declare conflict with each other = 2 conflict entries
	if len(conflicts) < 1 {
		t.Errorf("Expected at least 1 conflict, got %d", len(conflicts))
	}
}

func TestConflictDetector_ProvidesConflict(t *testing.T) {
	db := &conflictTestDB{
		packages: map[string]pkgdb.PackageInfo{
			"vim":    {Name: "vim", Provides: []string{"vi"}},
			"neovim": {Name: "neovim", Provides: []string{"vi"}},
		},
	}

	detector := resolver.NewConflictDetector(db)
	conflicts := detector.Detect([]string{"vim", "neovim"})

	// Both provide "vi" - this is a conflict
	if len(conflicts) != 1 {
		t.Errorf("Expected 1 provides conflict, got %d", len(conflicts))
	}
}

func TestConflict_String(t *testing.T) {
	c := resolver.Conflict{
		Package1: "vim",
		Package2: "neovim",
		Reason:   "both provide vi",
	}

	str := c.String()
	if str == "" {
		t.Error("Conflict.String() should not be empty")
	}
}

// conflictTestDB for testing
type conflictTestDB struct {
	packages map[string]pkgdb.PackageInfo
}

func (m *conflictTestDB) Search(query string) ([]pkgdb.PackageInfo, error) {
	var results []pkgdb.PackageInfo
	for _, p := range m.packages {
		results = append(results, p)
	}
	return results, nil
}

func (m *conflictTestDB) GetInfo(name string) (*pkgdb.PackageInfo, error) {
	if p, ok := m.packages[name]; ok {
		return &p, nil
	}
	return nil, nil
}
