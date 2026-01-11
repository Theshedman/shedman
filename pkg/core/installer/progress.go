package installer

import (
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/theshedman/shedman/pkg/backend"
)

// ProgressCallback is called with progress updates during download/install
type ProgressCallback func(event ProgressEvent)

// ProgressEvent represents a progress update
type ProgressEvent struct {
	Type       ProgressType
	Package    string
	Current    int    // Current package number
	Total      int    // Total packages
	Percentage int    // Download percentage (0-100)
	Speed      string // Download speed (e.g., "1.5 MiB/s")
	Downloaded string // Downloaded size (e.g., "5.2 MiB")
	TotalSize  string // Total size (e.g., "10.4 MiB")
	Message    string // Human-readable message
}

// ProgressType indicates the type of progress event
type ProgressType int

const (
	ProgressDownloading ProgressType = iota
	ProgressInstalling
	ProgressUpgrading
	ProgressRemoving
	ProgressComplete
	ProgressError
	ProgressResolving
	ProgressChecking
)

// String returns a string representation of ProgressType
func (pt ProgressType) String() string {
	switch pt {
	case ProgressDownloading:
		return "downloading"
	case ProgressInstalling:
		return "installing"
	case ProgressUpgrading:
		return "upgrading"
	case ProgressRemoving:
		return "removing"
	case ProgressComplete:
		return "complete"
	case ProgressError:
		return "error"
	case ProgressResolving:
		return "resolving"
	case ProgressChecking:
		return "checking"
	default:
		return "unknown"
	}
}

// Note: Pacman-specific progress parsing has been moved to backend/pacman/progress.go
// Use pacman.NewProgress() for parsing pacman output

// FileScanResult contains the results of a filesystem scan
type FileScanResult struct {
	Files       map[string]FileInfo // path -> info
	TotalFiles  int
	TotalSize   int64
	Directories []string
}

// FileInfo contains information about a file on disk
type FileInfo struct {
	Path  string
	Size  int64
	Mode  fs.FileMode
	IsDir bool
	Owner string // Package that owns this file (if known)
}

// FilesystemScanner scans the filesystem for existing files
type FilesystemScanner struct {
	root     string
	skipDirs []string
	maxDepth int
}

// NewFilesystemScanner creates a new filesystem scanner
func NewFilesystemScanner(root string) *FilesystemScanner {
	return &FilesystemScanner{
		root: root,
		skipDirs: []string{
			"/proc", "/sys", "/dev", "/run", "/tmp",
			"/var/cache", "/var/log", "/var/tmp",
			"/home", "/root",
		},
		maxDepth: 10,
	}
}

// SetSkipDirs sets directories to skip during scanning
func (fs *FilesystemScanner) SetSkipDirs(dirs []string) {
	fs.skipDirs = dirs
}

// SetMaxDepth sets maximum traversal depth
func (fs *FilesystemScanner) SetMaxDepth(depth int) {
	fs.maxDepth = depth
}

// shouldSkip checks if a path should be skipped
func (fsc *FilesystemScanner) shouldSkip(path string) bool {
	for _, skip := range fsc.skipDirs {
		if strings.HasPrefix(path, skip) {
			return true
		}
	}
	return false
}

// ScanPath scans a specific path and returns file info
func (fsc *FilesystemScanner) ScanPath(targetPath string) (*FileScanResult, error) {
	result := &FileScanResult{
		Files:       make(map[string]FileInfo),
		Directories: make([]string, 0),
	}

	fullPath := filepath.Join(fsc.root, targetPath)

	err := filepath.WalkDir(fullPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Skip permission errors
			return nil
		}

		// Get relative path from root
		relPath, _ := filepath.Rel(fsc.root, path)
		if relPath == "." {
			return nil
		}

		absPath := "/" + relPath

		// Skip excluded directories
		if d.IsDir() && fsc.shouldSkip(absPath) {
			return filepath.SkipDir
		}

		if d.IsDir() {
			result.Directories = append(result.Directories, absPath)
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		result.Files[absPath] = FileInfo{
			Path:  absPath,
			Size:  info.Size(),
			Mode:  info.Mode(),
			IsDir: false,
		}
		result.TotalFiles++
		result.TotalSize += info.Size()

		return nil
	})

	return result, err
}

// ScanPaths scans multiple paths
func (fsc *FilesystemScanner) ScanPaths(paths []string) (*FileScanResult, error) {
	combined := &FileScanResult{
		Files:       make(map[string]FileInfo),
		Directories: make([]string, 0),
	}

	for _, path := range paths {
		result, err := fsc.ScanPath(path)
		if err != nil {
			continue // Skip paths that fail
		}

		for k, v := range result.Files {
			combined.Files[k] = v
			combined.TotalFiles++
			combined.TotalSize += v.Size
		}
		combined.Directories = append(combined.Directories, result.Directories...)
	}

	return combined, nil
}

// GetPackageFilesWithBackend gets the list of files owned by a package using the backend
func GetPackageFilesWithBackend(b backend.OfficialBackend, packageName string) ([]string, error) {
	if b == nil {
		return nil, nil
	}
	return b.GetPackageFiles(packageName)
}

// GetInstalledPackageFilesWithBackend gets file lists for all installed packages using the backend
func GetInstalledPackageFilesWithBackend(b backend.OfficialBackend) (map[string][]string, error) {
	result := make(map[string][]string)

	if b == nil {
		return result, nil
	}

	// Get list of installed packages
	packages, err := b.GetInstalledPackages()
	if err != nil {
		return nil, err
	}

	for _, pkg := range packages {
		files, err := b.GetPackageFiles(pkg.Name)
		if err != nil {
			// Log but continue - some packages may not have file lists
			continue
		}
		result[pkg.Name] = files
	}

	return result, nil
}

// CheckFileConflicts checks if package files conflict with existing files
func CheckFileConflicts(packageFiles []string, existingFiles map[string]FileInfo) []string {
	conflicts := make([]string, 0)
	for _, file := range packageFiles {
		if _, exists := existingFiles[file]; exists {
			conflicts = append(conflicts, file)
		}
	}
	return conflicts
}
