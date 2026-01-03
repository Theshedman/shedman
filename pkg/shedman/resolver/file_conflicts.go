package resolver

import (
	"path/filepath"
	"sort"
)

// FileConflictType represents the type of file conflict
type FileConflictType int

const (
	FileConflictOwnership FileConflictType = iota // Two packages own the same file
	FileConflictExisting                          // Package file conflicts with existing file on disk
)

// FileConflict represents a file ownership conflict
type FileConflict struct {
	FilePath string
	Package1 string // First package (or empty for existing file)
	Package2 string // Second package
	Type     FileConflictType
}

// FileConflictChecker detects file-level conflicts between packages
type FileConflictChecker struct {
	// Map of file path to list of owning packages
	fileOwners        map[string][]string
	existingFiles     map[string]string // Existing files on disk (path -> owner or empty)
	overwritePatterns []string
}

// NewFileConflictChecker creates a new FileConflictChecker
func NewFileConflictChecker() *FileConflictChecker {
	return &FileConflictChecker{
		fileOwners:        make(map[string][]string),
		existingFiles:     make(map[string]string),
		overwritePatterns: make([]string, 0),
	}
}

// SetOverwritePattern adds a glob pattern for files that can be overwritten
func (fc *FileConflictChecker) SetOverwritePattern(pattern string) {
	fc.overwritePatterns = append(fc.overwritePatterns, pattern)
}

// SetOverwritePatterns sets multiple glob patterns
func (fc *FileConflictChecker) SetOverwritePatterns(patterns []string) {
	fc.overwritePatterns = patterns
}

// RegisterPackageFiles registers the files owned by a package
func (fc *FileConflictChecker) RegisterPackageFiles(pkgName string, files []string) {
	for _, file := range files {
		fc.fileOwners[file] = append(fc.fileOwners[file], pkgName)
	}
}

// RegisterExistingFile registers a file that already exists on disk
func (fc *FileConflictChecker) RegisterExistingFile(path, owner string) {
	fc.existingFiles[path] = owner
}

// CheckConflicts returns all detected file conflicts
func (fc *FileConflictChecker) CheckConflicts() []FileConflict {
	conflicts := make([]FileConflict, 0)
	seen := make(map[string]bool) // Track seen files for deduplication

	// Check for ownership conflicts (two packages own same file)
	for file, packages := range fc.fileOwners {
		if len(packages) > 1 {
			// Check if overwrite pattern allows this
			if fc.isOverwriteAllowed(file) {
				continue
			}

			// Sort for deterministic output
			sort.Strings(packages)
			conflicts = append(conflicts, FileConflict{
				FilePath: file,
				Package1: packages[0],
				Package2: packages[1],
				Type:     FileConflictOwnership,
			})
			seen[file] = true
		}
	}

	// Check for existing file conflicts
	for path := range fc.existingFiles {
		if seen[path] {
			continue
		}

		// Check if any package wants to install to this path
		if packages, exists := fc.fileOwners[path]; exists && len(packages) > 0 {
			// Check if overwrite pattern allows this
			if fc.isOverwriteAllowed(path) {
				continue
			}

			conflicts = append(conflicts, FileConflict{
				FilePath: path,
				Package1: "",
				Package2: packages[0],
				Type:     FileConflictExisting,
			})
		}
	}

	return conflicts
}

// isOverwriteAllowed checks if a file matches any overwrite pattern
func (fc *FileConflictChecker) isOverwriteAllowed(filePath string) bool {
	for _, pattern := range fc.overwritePatterns {
		// Try glob match
		matched, err := filepath.Match(pattern, filePath)
		if err == nil && matched {
			return true
		}

		// Also try matching just the directory portion
		dir := filepath.Dir(filePath)
		if dir+"/*" == pattern {
			return true
		}
	}
	return false
}

// HasConflicts returns true if there are any conflicts
func (fc *FileConflictChecker) HasConflicts() bool {
	return len(fc.CheckConflicts()) > 0
}

// GetConflictingFiles returns just the file paths that have conflicts
func (fc *FileConflictChecker) GetConflictingFiles() []string {
	conflicts := fc.CheckConflicts()
	files := make([]string, len(conflicts))
	for i, c := range conflicts {
		files[i] = c.FilePath
	}
	return files
}
