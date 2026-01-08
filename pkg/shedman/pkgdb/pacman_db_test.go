package pkgdb_test

import (
	"errors"
	"testing"

	"github.com/theshedman/shedman/pkg/shedman/pkgdb"
)

// MockBackend implements pkgdb.PackageSearcher for testing
type MockBackend struct {
	SearchFunc      func(query string) ([]pkgdb.PackageInfo, error)
	InfoFunc        func(name string) (*pkgdb.PackageInfo, error)
	IsInstalledFunc func(name string) bool
	IsAvailableFunc func() bool
}

func (m *MockBackend) Search(query string) ([]pkgdb.PackageInfo, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(query)
	}
	return nil, nil
}

func (m *MockBackend) Info(name string) (*pkgdb.PackageInfo, error) {
	if m.InfoFunc != nil {
		return m.InfoFunc(name)
	}
	return nil, errors.New("not found")
}

func (m *MockBackend) IsInstalled(name string) bool {
	if m.IsInstalledFunc != nil {
		return m.IsInstalledFunc(name)
	}
	return false
}

func (m *MockBackend) IsAvailable() bool {
	if m.IsAvailableFunc != nil {
		return m.IsAvailableFunc()
	}
	return true
}

func TestPacmanDB_NewPacmanDB(t *testing.T) {
	db := pkgdb.NewPacmanDB()
	if db == nil {
		t.Fatal("NewPacmanDB should return non-nil")
	}
}

func TestPacmanDB_ImplementsPackageDB(t *testing.T) {
	db := pkgdb.NewPacmanDB()
	var _ pkgdb.PackageDB = db
}

func TestPacmanDB_Search(t *testing.T) {
	mock := &MockBackend{
		SearchFunc: func(query string) ([]pkgdb.PackageInfo, error) {
			return []pkgdb.PackageInfo{
				{Name: "neovim", Version: "0.10.0-1", Description: "Vim fork"},
			}, nil
		},
	}
	db := pkgdb.NewPacmanDBWithBackend(mock)

	results, err := db.Search("neovim")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
	if results[0].Name != "neovim" {
		t.Errorf("Expected name 'neovim', got '%s'", results[0].Name)
	}
}

func TestPacmanDB_GetInfo(t *testing.T) {
	mock := &MockBackend{
		InfoFunc: func(name string) (*pkgdb.PackageInfo, error) {
			return &pkgdb.PackageInfo{
				Name:        "neovim",
				Version:     "0.10.0-1",
				Description: "Vim fork focused on extensibility",
				Depends:     []string{"luajit", "msgpack-c"},
			}, nil
		},
	}
	db := pkgdb.NewPacmanDBWithBackend(mock)

	info, err := db.GetInfo("neovim")
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}

	if info == nil {
		t.Fatal("Expected package info, got nil")
	}
	if info.Name != "neovim" {
		t.Errorf("Expected name 'neovim', got '%s'", info.Name)
	}
	if info.Version != "0.10.0-1" {
		t.Errorf("Expected version '0.10.0-1', got '%s'", info.Version)
	}
	if len(info.Depends) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(info.Depends))
	}
}

func TestPacmanDB_GetInfo_NotFound(t *testing.T) {
	mock := &MockBackend{
		InfoFunc: func(name string) (*pkgdb.PackageInfo, error) {
			return nil, nil
		},
	}
	db := pkgdb.NewPacmanDBWithBackend(mock)

	info, err := db.GetInfo("nonexistent")
	if err != nil {
		t.Fatalf("GetInfo should not error for not found: %v", err)
	}

	if info != nil {
		t.Error("Expected nil for non-existent package")
	}
}

func TestPacmanDB_IsInstalled(t *testing.T) {
	mock := &MockBackend{
		IsInstalledFunc: func(name string) bool {
			return name == "neovim"
		},
	}
	db := pkgdb.NewPacmanDBWithBackend(mock)

	if !db.IsInstalled("neovim") {
		t.Error("Expected neovim to be installed")
	}
	if db.IsInstalled("vim") {
		t.Error("Expected vim to not be installed")
	}
}

func TestPacmanDB_IsAlpmAvailable(t *testing.T) {
	mock := &MockBackend{
		IsAvailableFunc: func() bool { return true },
	}
	db := pkgdb.NewPacmanDBWithBackend(mock)

	if !db.IsAlpmAvailable() {
		t.Error("Expected alpm to be available")
	}
}

func TestPacmanDB_NilBackend(t *testing.T) {
	db := pkgdb.NewPacmanDBWithBackend(nil)

	if db.IsAlpmAvailable() {
		t.Error("Expected alpm to not be available with nil backend")
	}

	_, err := db.Search("test")
	if err != pkgdb.ErrPacmanNotFound {
		t.Errorf("Expected ErrPacmanNotFound, got %v", err)
	}
}

func TestPacmanDB_SetBackend(t *testing.T) {
	db := pkgdb.NewPacmanDB()

	// Initially nil
	if db.IsAlpmAvailable() {
		t.Error("Expected not available initially")
	}

	// Set backend
	mock := &MockBackend{IsAvailableFunc: func() bool { return true }}
	db.SetBackend(mock)

	if !db.IsAlpmAvailable() {
		t.Error("Expected available after SetBackend")
	}
}

func TestPacmanDB_ParsePacmanInfo(t *testing.T) {
	output := `Name            : git
Version         : 2.43.0-1
Description     : The fast distributed version control system
Depends On      : curl  expat  perl  openssl
Optional Deps   : libsecret: credential storage
Provides        : git-core
Conflicts       : git-core
Download Size   : 8.5 MiB
Installed Size  : 35.0 MiB
`
	info := pkgdb.ParsePacmanInfo(output)

	if info.Name != "git" {
		t.Errorf("Expected name 'git', got '%s'", info.Name)
	}
	if info.Version != "2.43.0-1" {
		t.Errorf("Expected version '2.43.0-1', got '%s'", info.Version)
	}
	if len(info.Depends) != 4 {
		t.Errorf("Expected 4 dependencies, got %d", len(info.Depends))
	}
	if len(info.Provides) != 1 || info.Provides[0] != "git-core" {
		t.Errorf("Expected provides [git-core], got %v", info.Provides)
	}
	if len(info.Conflicts) != 1 || info.Conflicts[0] != "git-core" {
		t.Errorf("Expected conflicts [git-core], got %v", info.Conflicts)
	}

	expectedSize := int64(8912896)
	if info.Size != expectedSize {
		t.Errorf("Expected download size %d, got %d", expectedSize, info.Size)
	}

	expectedInstalled := int64(36700160)
	if info.InstalledSize != expectedInstalled {
		t.Errorf("Expected installed size %d, got %d", expectedInstalled, info.InstalledSize)
	}
}

func TestPacmanDB_ParseSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"10.5 MiB", 11010048},
		{"8.5 MiB", 8912896},
		{"35.0 MiB", 36700160},
		{"1.0 GiB", 1073741824},
		{"512 KiB", 524288},
		{"100 B", 100},
		{"", 0},
		{"invalid", 0},
	}

	for _, tc := range tests {
		result := pkgdb.ParseSize(tc.input)
		if result != tc.expected {
			t.Errorf("ParseSize(%q) = %d, expected %d", tc.input, result, tc.expected)
		}
	}
}
