package alpm

import (
	"testing"
)

// --- Tests ---

func TestMockAlpmDB_Pkg(t *testing.T) {
	mockPkg := &MockAlpmPackage{NameVal: "testpkg", VersionVal: "1.0.0"}
	mockDB := &MockAlpmDB{
		NameVal:  "testdb",
		Packages: map[string]AlpmPackage{"testpkg": mockPkg},
	}

	pkg := mockDB.Pkg("testpkg")
	if pkg == nil {
		t.Fatal("Expected package, got nil")
	}
	if pkg.Name() != "testpkg" {
		t.Errorf("Expected 'testpkg', got '%s'", pkg.Name())
	}

	notFound := mockDB.Pkg("nonexistent")
	if notFound != nil {
		t.Error("Expected nil for nonexistent package")
	}
}

func TestMockAlpmDB_Search(t *testing.T) {
	vimPkg := &MockAlpmPackage{NameVal: "pkg1", VersionVal: "1.0"}
	neovimPkg := &MockAlpmPackage{NameVal: "pkg2", VersionVal: "2.0"}

	mockDB := &MockAlpmDB{
		NameVal: "testdb",
		SearchFn: func(targets []string) AlpmPackageList {
			// Simple mock: return both packages for any search
			return &MockAlpmPackageList{Packages: []AlpmPackage{vimPkg, neovimPkg}}
		},
	}

	results := mockDB.Search([]string{"vim"})
	if results.Len() != 2 {
		t.Errorf("Expected 2 results, got %d", results.Len())
	}
}

func TestMockAlpmDBList_ForEach(t *testing.T) {
	db1 := &MockAlpmDB{NameVal: "core"}
	db2 := &MockAlpmDB{NameVal: "extra"}

	dbList := &MockAlpmDBList{Dbs: []AlpmDB{db1, db2}}

	var names []string
	err := dbList.ForEach(func(db AlpmDB) error {
		names = append(names, db.Name())
		return nil
	})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(names) != 2 {
		t.Errorf("Expected 2 databases, got %d", len(names))
	}
}

func TestMockAlpmHandle_Release(t *testing.T) {
	handle := &MockAlpmHandle{}

	if handle.Released {
		t.Error("Handle should not be released initially")
	}

	err := handle.Release()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !handle.Released {
		t.Error("Handle should be released after Release()")
	}
}
