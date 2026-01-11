package core

import (
	"testing"

)

func TestConflictDetector_NoConflicts(t *testing.T) {
	db := &conflictTestDB{
		packages: map[string]PackageInfo{
			"neovim": {Name: "neovim"},
			"git":    {Name: "git"},
		},
	}

	detector := NewConflictDetector(db)
	conflicts := detector.Detect([]string{"neovim", "git"})

	if len(conflicts) != 0 {
		t.Errorf("Expected no conflicts, got %v", conflicts)
	}
}

func TestConflictDetector_PackageConflict(t *testing.T) {
	db := &conflictTestDB{
		packages: map[string]PackageInfo{
			"vim":    {Name: "vim", Conflicts: []string{"neovim"}},
			"neovim": {Name: "neovim", Conflicts: []string{"vim"}},
		},
	}

	detector := NewConflictDetector(db)
	conflicts := detector.Detect([]string{"vim", "neovim"})

	// Both packages declare conflict with each other = 2 conflict entries
	if len(conflicts) < 1 {
		t.Errorf("Expected at least 1 conflict, got %d", len(conflicts))
	}
}

func TestConflictDetector_ProvidesConflict(t *testing.T) {
	db := &conflictTestDB{
		packages: map[string]PackageInfo{
			"vim":    {Name: "vim", Provides: []string{"vi"}},
			"neovim": {Name: "neovim", Provides: []string{"vi"}},
		},
	}

	detector := NewConflictDetector(db)
	conflicts := detector.Detect([]string{"vim", "neovim"})

	// Both provide "vi" - this is a conflict
	if len(conflicts) != 1 {
		t.Errorf("Expected 1 provides conflict, got %d", len(conflicts))
	}
}

func TestConflict_String(t *testing.T) {
	c := Conflict{
		Package1: "vim",
		Package2: "neovim",
		Reason:   "both provide vi",
	}

	str := c.String()
	if str == "" {
		t.Error("Conflict.String() should not be empty")
	}
}

// ============== Phase 4 Refactoring Tests ==============

func TestConflictDetector_VersionAwareConflict(t *testing.T) {
	db := &conflictTestDB{
		packages: map[string]PackageInfo{
			"app": {Name: "app", Version: "1.0", Conflicts: []string{"lib>=2.0"}},
			"lib": {Name: "lib", Version: "2.5.0"},
		},
	}

	detector := NewConflictDetector(db)
	conflicts := detector.Detect([]string{"app", "lib"})

	// lib 2.5.0 >= 2.0, so conflict should be detected
	if len(conflicts) != 1 {
		t.Errorf("Expected 1 conflict (lib>=2.0 matches 2.5.0), got %d", len(conflicts))
	}
}

func TestConflictDetector_VersionNoConflict(t *testing.T) {
	db := &conflictTestDB{
		packages: map[string]PackageInfo{
			"app": {Name: "app", Version: "1.0", Conflicts: []string{"lib>=3.0"}},
			"lib": {Name: "lib", Version: "2.5.0"},
		},
	}

	detector := NewConflictDetector(db)
	conflicts := detector.Detect([]string{"app", "lib"})

	// lib 2.5.0 < 3.0, so no conflict
	if len(conflicts) != 0 {
		t.Errorf("Expected no conflicts (lib 2.5.0 < 3.0), got %d", len(conflicts))
	}
}

func TestConflictDetector_WithInstalled(t *testing.T) {
	newDB := &conflictTestDB{
		packages: map[string]PackageInfo{
			"neovim": {Name: "neovim", Version: "0.10.0", Conflicts: []string{"vim"}},
		},
	}
	installedDB := &conflictTestDB{
		packages: map[string]PackageInfo{
			"vim": {Name: "vim", Version: "9.0"},
		},
	}

	detector := NewConflictDetectorWithInstalled(newDB, installedDB)
	conflicts := detector.DetectWithInstalled([]string{"neovim"}, []string{"vim"})

	if len(conflicts) != 1 {
		t.Errorf("Expected 1 conflict with installed vim, got %d", len(conflicts))
	}
	if len(conflicts) > 0 && conflicts[0].Reason != "conflicts with installed package" {
		t.Errorf("Expected 'conflicts with installed package', got %s", conflicts[0].Reason)
	}
}

func TestConflictResult_HasErrors(t *testing.T) {
	result := &ConflictResult{
		PackageConflicts: []Conflict{
			{Package1: "a", Package2: "b", Severity: ConflictError},
		},
	}

	if !result.HasErrors() {
		t.Error("Expected HasErrors to return true")
	}
}

func TestConflictResult_IsEmpty(t *testing.T) {
	emptyResult := &ConflictResult{}
	if !emptyResult.IsEmpty() {
		t.Error("Expected empty result")
	}

	nonEmptyResult := &ConflictResult{
		PackageConflicts: []Conflict{{Package1: "a", Package2: "b"}},
	}
	if nonEmptyResult.IsEmpty() {
		t.Error("Expected non-empty result")
	}
}

func TestFileConflict_OwnershipConflict(t *testing.T) {
	fc := NewFileConflictChecker()

	// Simulate two packages owning the same file
	fc.RegisterPackageFiles("vim", []string{"/usr/bin/vi", "/usr/share/vim/vimrc"})
	fc.RegisterPackageFiles("neovim", []string{"/usr/bin/vi", "/usr/share/nvim/init.vim"})

	conflicts := fc.CheckConflicts()

	if len(conflicts) != 1 {
		t.Errorf("Expected 1 file conflict for /usr/bin/vi, got %d", len(conflicts))
	}
	if len(conflicts) > 0 && conflicts[0].FilePath != "/usr/bin/vi" {
		t.Errorf("Expected conflict on /usr/bin/vi, got %s", conflicts[0].FilePath)
	}
}

func TestFileConflict_NoConflict(t *testing.T) {
	fc := NewFileConflictChecker()

	fc.RegisterPackageFiles("git", []string{"/usr/bin/git"})
	fc.RegisterPackageFiles("curl", []string{"/usr/bin/curl"})

	conflicts := fc.CheckConflicts()

	if len(conflicts) != 0 {
		t.Errorf("Expected no file conflicts, got %d", len(conflicts))
	}
}

func TestFileConflict_WithOverwrite(t *testing.T) {
	fc := NewFileConflictChecker()
	fc.SetOverwritePattern("/usr/bin/*")

	fc.RegisterPackageFiles("vim", []string{"/usr/bin/vi"})
	fc.RegisterPackageFiles("neovim", []string{"/usr/bin/vi"})

	conflicts := fc.CheckConflicts()

	// Overwrite pattern should suppress the conflict
	if len(conflicts) != 0 {
		t.Errorf("Expected overwrite pattern to suppress conflict, got %d", len(conflicts))
	}
}

func TestFileConflict_Type(t *testing.T) {
	fc := FileConflict{
		FilePath: "/usr/bin/vi",
		Package1: "vim",
		Package2: "neovim",
		Type:     FileConflictOwnership,
	}

	if fc.Type != FileConflictOwnership {
		t.Error("Expected ownership conflict type")
	}
}

func TestFileConflict_ExistingFileOnDisk(t *testing.T) {
	fc := NewFileConflictChecker()

	// Register an "existing" file (simulates file already on disk)
	fc.RegisterExistingFile("/etc/config.conf", "")

	// New package wants to install to same path
	fc.RegisterPackageFiles("myapp", []string{"/etc/config.conf"})

	conflicts := fc.CheckConflicts()

	if len(conflicts) != 1 {
		t.Errorf("Expected 1 conflict with existing file, got %d", len(conflicts))
	}
	if len(conflicts) > 0 && conflicts[0].Type != FileConflictExisting {
		t.Errorf("Expected existing file conflict type, got %v", conflicts[0].Type)
	}
}

// conflictTestDB for testing
type conflictTestDB struct {
	packages map[string]PackageInfo
}

func (m *conflictTestDB) Search(query string) ([]PackageInfo, error) {
	var results []PackageInfo
	for _, p := range m.packages {
		results = append(results, p)
	}
	return results, nil
}

func (m *conflictTestDB) GetInfo(name string) (*PackageInfo, error) {
	if p, ok := m.packages[name]; ok {
		return &p, nil
	}
	return nil, nil
}
