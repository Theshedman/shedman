package installer

import (
	"bufio"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
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

// PacmanProgress parses pacman output and emits progress events
type PacmanProgress struct {
	callback       ProgressCallback
	downloadRegex  *regexp.Regexp
	installRegex   *regexp.Regexp
	upgradeRegex   *regexp.Regexp
	percentRegex   *regexp.Regexp
	speedRegex     *regexp.Regexp
	progressRegex  *regexp.Regexp
	currentPkg     string
	totalPackages  int
	currentPackage int
}

// NewPacmanProgress creates a new progress parser
func NewPacmanProgress(callback ProgressCallback) *PacmanProgress {
	return &PacmanProgress{
		callback:      callback,
		downloadRegex: regexp.MustCompile(`(?i)downloading\s+(.+?)\.\.\.`),
		installRegex:  regexp.MustCompile(`(?i)installing\s+(.+?)\.\.\.`),
		upgradeRegex:  regexp.MustCompile(`(?i)upgrading\s+(.+?)\.\.\.`),
		percentRegex:  regexp.MustCompile(`\((\d+)/(\d+)\)\s+(\d+)%`),
		speedRegex:    regexp.MustCompile(`(\d+\.?\d*)\s*(B|KiB|MiB|GiB)/s`),
		progressRegex: regexp.MustCompile(`(\d+\.?\d*)\s*(B|KiB|MiB|GiB)\s*/\s*(\d+\.?\d*)\s*(B|KiB|MiB|GiB)`),
	}
}

// SetTotalPackages sets the expected total number of packages
func (pp *PacmanProgress) SetTotalPackages(total int) {
	pp.totalPackages = total
}

// ParseLine parses a line of pacman output and emits progress events
func (pp *PacmanProgress) ParseLine(line string) {
	if pp.callback == nil {
		return
	}

	line = strings.TrimSpace(line)
	if line == "" {
		return
	}

	event := ProgressEvent{
		Total:   pp.totalPackages,
		Current: pp.currentPackage,
	}

	// Parse percentage if present
	if matches := pp.percentRegex.FindStringSubmatch(line); len(matches) > 3 {
		event.Current, _ = strconv.Atoi(matches[1])
		event.Total, _ = strconv.Atoi(matches[2])
		event.Percentage, _ = strconv.Atoi(matches[3])
		pp.currentPackage = event.Current
		pp.totalPackages = event.Total
	}

	// Parse speed if present
	if matches := pp.speedRegex.FindStringSubmatch(line); len(matches) > 2 {
		event.Speed = matches[1] + " " + matches[2] + "/s"
	}

	// Parse download progress (e.g., "5.2 MiB / 10.4 MiB")
	if matches := pp.progressRegex.FindStringSubmatch(line); len(matches) > 4 {
		event.Downloaded = matches[1] + " " + matches[2]
		event.TotalSize = matches[3] + " " + matches[4]
	}

	// Check for downloading
	if matches := pp.downloadRegex.FindStringSubmatch(line); len(matches) > 1 {
		event.Type = ProgressDownloading
		event.Package = matches[1]
		event.Message = fmt.Sprintf("Downloading %s...", matches[1])
		pp.currentPkg = matches[1]
		pp.callback(event)
		return
	}

	// Check for installing
	if matches := pp.installRegex.FindStringSubmatch(line); len(matches) > 1 {
		event.Type = ProgressInstalling
		event.Package = matches[1]
		event.Message = fmt.Sprintf("Installing %s...", matches[1])
		pp.currentPkg = matches[1]
		pp.callback(event)
		return
	}

	// Check for upgrading
	if matches := pp.upgradeRegex.FindStringSubmatch(line); len(matches) > 1 {
		event.Type = ProgressUpgrading
		event.Package = matches[1]
		event.Message = fmt.Sprintf("Upgrading %s...", matches[1])
		pp.currentPkg = matches[1]
		pp.callback(event)
		return
	}

	// Check for removing
	if strings.Contains(strings.ToLower(line), "removing") {
		event.Type = ProgressRemoving
		event.Package = pp.currentPkg
		event.Message = line
		pp.callback(event)
		return
	}

	// Check for resolving dependencies
	if strings.Contains(strings.ToLower(line), "resolving") {
		event.Type = ProgressResolving
		event.Message = line
		pp.callback(event)
		return
	}

	// Check for checking
	if strings.Contains(strings.ToLower(line), "checking") {
		event.Type = ProgressChecking
		event.Message = line
		pp.callback(event)
		return
	}
}

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

// GetPacmanFiles gets the list of files owned by a package using pacman -Ql
func GetPacmanFiles(packageName string) ([]string, error) {
	cmd := exec.Command("pacman", "-Ql", packageName)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	files := make([]string, 0)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		// Format: "package /path/to/file"
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			path := strings.TrimSpace(parts[1])
			if path != "" && !strings.HasSuffix(path, "/") {
				files = append(files, path)
			}
		}
	}

	return files, nil
}

// GetInstalledPackageFiles gets file lists for all installed packages
func GetInstalledPackageFiles() (map[string][]string, error) {
	result := make(map[string][]string)

	// Get list of installed packages
	cmd := exec.Command("pacman", "-Q")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) >= 1 {
			pkgName := parts[0]
			files, err := GetPacmanFiles(pkgName)
			if err == nil {
				result[pkgName] = files
			}
		}
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
