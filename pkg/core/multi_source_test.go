package core

import (
	"errors"
	"testing"
)

// mockDB implements PackageDB for testing multi-source resolver
type multiSourceMockDB struct {
	packages map[string]*PackageInfo
	source   string
}

func (m *multiSourceMockDB) Search(query string) ([]PackageInfo, error) {
	return nil, nil
}

func (m *multiSourceMockDB) GetInfo(name string) (*PackageInfo, error) {
	if info, ok := m.packages[name]; ok {
		return info, nil
	}
	return nil, nil
}

type errorMockDB struct {
	err error
}

func (e *errorMockDB) Search(query string) ([]PackageInfo, error) {
	return nil, e.err
}

func (e *errorMockDB) GetInfo(name string) (*PackageInfo, error) {
	return nil, e.err
}

func TestMultiSourceResolver_New(t *testing.T) {
	r := NewMultiSource()
	if r == nil {
		t.Fatal("NewMultiSource should return non-nil")
	}
}

func TestMultiSourceResolver_AddSource(t *testing.T) {
	r := NewMultiSource()

	shedDB := &multiSourceMockDB{source: SourceShedOS, packages: map[string]*PackageInfo{}}
	r.AddSource(SourceShedOS, shedDB)

	// No error = success
}

func TestMultiSourceResolver_FindPackage_Priority(t *testing.T) {
	r := NewMultiSource()

	// ShedOS DB has the package
	shedDB := &multiSourceMockDB{
		source: SourceShedOS,
		packages: map[string]*PackageInfo{
			"test-pkg": {Name: "test-pkg", Source: SourceShedOS},
		},
	}

	// Official DB also has the package
	officialDB := &multiSourceMockDB{
		source: SourceOfficial,
		packages: map[string]*PackageInfo{
			"test-pkg": {Name: "test-pkg", Source: SourceOfficial},
		},
	}

	// AUR DB also has the package
	aurDB := &multiSourceMockDB{
		source: SourceAUR,
		packages: map[string]*PackageInfo{
			"test-pkg": {Name: "test-pkg", Source: SourceAUR},
		},
	}

	// Add in priority order: ShedOS → Official → AUR
	r.AddSource(SourceShedOS, shedDB)
	r.AddSource(SourceOfficial, officialDB)
	r.AddSource(SourceAUR, aurDB)

	// Should find from ShedOS first (highest priority)
	info, err := r.FindPackage("test-pkg", "")
	if err != nil {
		t.Fatalf("FindPackage failed: %v", err)
	}
	if info == nil {
		t.Fatal("Expected to find package")
	}
	if info.Source != SourceShedOS {
		t.Errorf("Expected source ShedOS, got %s", info.Source)
	}
}

func TestMultiSourceResolver_FindPackage_ForcedSource(t *testing.T) {
	r := NewMultiSource()

	shedDB := &multiSourceMockDB{
		source: SourceShedOS,
		packages: map[string]*PackageInfo{
			"test-pkg": {Name: "test-pkg", Source: SourceShedOS},
		},
	}

	aurDB := &multiSourceMockDB{
		source: SourceAUR,
		packages: map[string]*PackageInfo{
			"test-pkg": {Name: "test-pkg", Source: SourceAUR},
		},
	}

	r.AddSource(SourceShedOS, shedDB)
	r.AddSource(SourceAUR, aurDB)

	// Force AUR source
	info, err := r.FindPackage("test-pkg", SourceAUR)
	if err != nil {
		t.Fatalf("FindPackage failed: %v", err)
	}
	if info.Source != SourceAUR {
		t.Errorf("Expected forced source AUR, got %s", info.Source)
	}
}

func TestMultiSourceResolver_FindPackage_Fallback(t *testing.T) {
	r := NewMultiSource()

	// ShedOS DB does NOT have the package
	shedDB := &multiSourceMockDB{
		source:   SourceShedOS,
		packages: map[string]*PackageInfo{},
	}

	// Official DB has the package
	officialDB := &multiSourceMockDB{
		source: SourceOfficial,
		packages: map[string]*PackageInfo{
			"test-pkg": {Name: "test-pkg", Source: SourceOfficial},
		},
	}

	r.AddSource(SourceShedOS, shedDB)
	r.AddSource(SourceOfficial, officialDB)

	// Should fallback to Official since not in ShedOS
	info, err := r.FindPackage("test-pkg", "")
	if err != nil {
		t.Fatalf("FindPackage failed: %v", err)
	}
	if info.Source != SourceOfficial {
		t.Errorf("Expected fallback to official, got %s", info.Source)
	}
}

func TestMultiSourceResolver_FindPackage_NotFound(t *testing.T) {
	r := NewMultiSource()

	shedDB := &multiSourceMockDB{packages: map[string]*PackageInfo{}}
	officialDB := &multiSourceMockDB{packages: map[string]*PackageInfo{}}

	r.AddSource(SourceShedOS, shedDB)
	r.AddSource(SourceOfficial, officialDB)

	info, err := r.FindPackage("nonexistent", "")
	if err != nil {
		t.Fatalf("FindPackage should not error: %v", err)
	}
	if info != nil {
		t.Error("Expected nil for non-existent package")
	}
}

func TestMultiSourceResolver_FindPackage_ForcedSourceMissing(t *testing.T) {
	r := NewMultiSource()

	_, err := r.FindPackage("test-pkg", SourceAUR)
	if err == nil {
		t.Fatal("Expected error when forced source is missing")
	}
}

func TestMultiSourceResolver_FindPackage_AllSourcesError(t *testing.T) {
	r := NewMultiSource()

	r.AddSource(SourceShedOS, &errorMockDB{err: errors.New("shedos down")})
	r.AddSource(SourceOfficial, &errorMockDB{err: errors.New("official down")})

	_, err := r.FindPackage("test-pkg", "")
	if err == nil {
		t.Fatal("Expected error when all sources fail")
	}
}

func TestMultiSourceResolver_Search_AllSourcesError(t *testing.T) {
	r := NewMultiSource()

	r.AddSource(SourceShedOS, &errorMockDB{err: errors.New("shedos down")})
	r.AddSource(SourceOfficial, &errorMockDB{err: errors.New("official down")})

	_, err := r.Search("test-pkg")
	if err == nil {
		t.Fatal("Expected error when all sources fail")
	}
}
