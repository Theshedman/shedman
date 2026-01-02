package resolver_test

import (
	"testing"

	"github.com/theshedman/shedman/pkg/shedman/pkgdb"
	"github.com/theshedman/shedman/pkg/shedman/resolver"
)

// mockDB implements pkgdb.PackageDB for testing multi-source resolver
type multiSourceMockDB struct {
	packages map[string]*pkgdb.PackageInfo
	source   string
}

func (m *multiSourceMockDB) Search(query string) ([]pkgdb.PackageInfo, error) {
	return nil, nil
}

func (m *multiSourceMockDB) GetInfo(name string) (*pkgdb.PackageInfo, error) {
	if info, ok := m.packages[name]; ok {
		return info, nil
	}
	return nil, nil
}

func TestMultiSourceResolver_New(t *testing.T) {
	r := resolver.NewMultiSource()
	if r == nil {
		t.Fatal("NewMultiSource should return non-nil")
	}
}

func TestMultiSourceResolver_AddSource(t *testing.T) {
	r := resolver.NewMultiSource()

	shedDB := &multiSourceMockDB{source: pkgdb.SourceShedOS, packages: map[string]*pkgdb.PackageInfo{}}
	r.AddSource(pkgdb.SourceShedOS, shedDB)

	// No error = success
}

func TestMultiSourceResolver_FindPackage_Priority(t *testing.T) {
	r := resolver.NewMultiSource()

	// ShedOS DB has the package
	shedDB := &multiSourceMockDB{
		source: pkgdb.SourceShedOS,
		packages: map[string]*pkgdb.PackageInfo{
			"test-pkg": {Name: "test-pkg", Source: pkgdb.SourceShedOS},
		},
	}

	// Official DB also has the package
	officialDB := &multiSourceMockDB{
		source: pkgdb.SourceOfficial,
		packages: map[string]*pkgdb.PackageInfo{
			"test-pkg": {Name: "test-pkg", Source: pkgdb.SourceOfficial},
		},
	}

	// AUR DB also has the package
	aurDB := &multiSourceMockDB{
		source: pkgdb.SourceAUR,
		packages: map[string]*pkgdb.PackageInfo{
			"test-pkg": {Name: "test-pkg", Source: pkgdb.SourceAUR},
		},
	}

	// Add in priority order: ShedOS → Official → AUR
	r.AddSource(pkgdb.SourceShedOS, shedDB)
	r.AddSource(pkgdb.SourceOfficial, officialDB)
	r.AddSource(pkgdb.SourceAUR, aurDB)

	// Should find from ShedOS first (highest priority)
	info, err := r.FindPackage("test-pkg", "")
	if err != nil {
		t.Fatalf("FindPackage failed: %v", err)
	}
	if info == nil {
		t.Fatal("Expected to find package")
	}
	if info.Source != pkgdb.SourceShedOS {
		t.Errorf("Expected source ShedOS, got %s", info.Source)
	}
}

func TestMultiSourceResolver_FindPackage_ForcedSource(t *testing.T) {
	r := resolver.NewMultiSource()

	shedDB := &multiSourceMockDB{
		source: pkgdb.SourceShedOS,
		packages: map[string]*pkgdb.PackageInfo{
			"test-pkg": {Name: "test-pkg", Source: pkgdb.SourceShedOS},
		},
	}

	aurDB := &multiSourceMockDB{
		source: pkgdb.SourceAUR,
		packages: map[string]*pkgdb.PackageInfo{
			"test-pkg": {Name: "test-pkg", Source: pkgdb.SourceAUR},
		},
	}

	r.AddSource(pkgdb.SourceShedOS, shedDB)
	r.AddSource(pkgdb.SourceAUR, aurDB)

	// Force AUR source
	info, err := r.FindPackage("test-pkg", pkgdb.SourceAUR)
	if err != nil {
		t.Fatalf("FindPackage failed: %v", err)
	}
	if info.Source != pkgdb.SourceAUR {
		t.Errorf("Expected forced source AUR, got %s", info.Source)
	}
}

func TestMultiSourceResolver_FindPackage_Fallback(t *testing.T) {
	r := resolver.NewMultiSource()

	// ShedOS DB does NOT have the package
	shedDB := &multiSourceMockDB{
		source:   pkgdb.SourceShedOS,
		packages: map[string]*pkgdb.PackageInfo{},
	}

	// Official DB has the package
	officialDB := &multiSourceMockDB{
		source: pkgdb.SourceOfficial,
		packages: map[string]*pkgdb.PackageInfo{
			"test-pkg": {Name: "test-pkg", Source: pkgdb.SourceOfficial},
		},
	}

	r.AddSource(pkgdb.SourceShedOS, shedDB)
	r.AddSource(pkgdb.SourceOfficial, officialDB)

	// Should fallback to Official since not in ShedOS
	info, err := r.FindPackage("test-pkg", "")
	if err != nil {
		t.Fatalf("FindPackage failed: %v", err)
	}
	if info.Source != pkgdb.SourceOfficial {
		t.Errorf("Expected fallback to official, got %s", info.Source)
	}
}

func TestMultiSourceResolver_FindPackage_NotFound(t *testing.T) {
	r := resolver.NewMultiSource()

	shedDB := &multiSourceMockDB{packages: map[string]*pkgdb.PackageInfo{}}
	officialDB := &multiSourceMockDB{packages: map[string]*pkgdb.PackageInfo{}}

	r.AddSource(pkgdb.SourceShedOS, shedDB)
	r.AddSource(pkgdb.SourceOfficial, officialDB)

	info, err := r.FindPackage("nonexistent", "")
	if err != nil {
		t.Fatalf("FindPackage should not error: %v", err)
	}
	if info != nil {
		t.Error("Expected nil for non-existent package")
	}
}
