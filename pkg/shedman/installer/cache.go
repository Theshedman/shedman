package installer

import (
	"bufio"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// PackageFileCache provides thread-safe caching of package file ownership
type PackageFileCache struct {
	mu         sync.RWMutex
	files      map[string][]string // package name -> file list
	fileOwners map[string]string   // file path -> package name
	lastUpdate time.Time
	ttl        time.Duration
}

// NewPackageFileCache creates a new cache with the specified TTL
func NewPackageFileCache(ttl time.Duration) *PackageFileCache {
	return &PackageFileCache{
		files:      make(map[string][]string),
		fileOwners: make(map[string]string),
		ttl:        ttl,
	}
}

// DefaultPackageFileCache creates a cache with 5-minute TTL
func DefaultPackageFileCache() *PackageFileCache {
	return NewPackageFileCache(5 * time.Minute)
}

// IsValid checks if the cache is still valid
func (c *PackageFileCache) IsValid() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.lastUpdate.IsZero() {
		return false
	}
	return time.Since(c.lastUpdate) < c.ttl
}

// Invalidate clears the cache
func (c *PackageFileCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.files = make(map[string][]string)
	c.fileOwners = make(map[string]string)
	c.lastUpdate = time.Time{}
}

// GetPackageFiles returns files for a package, using cache if valid
func (c *PackageFileCache) GetPackageFiles(packageName string) ([]string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	files, exists := c.files[packageName]
	return files, exists && c.IsValid()
}

// GetFileOwner returns the package that owns a file
func (c *PackageFileCache) GetFileOwner(filePath string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	owner, exists := c.fileOwners[filePath]
	return owner, exists && c.IsValid()
}

// SetPackageFiles stores files for a package
func (c *PackageFileCache) SetPackageFiles(packageName string, files []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.files[packageName] = files
	for _, f := range files {
		c.fileOwners[f] = packageName
	}
	c.lastUpdate = time.Now()
}

// LoadAll loads all installed package files using optimized batch query
func (c *PackageFileCache) LoadAll() error {
	// Use pacman -Qlq for quiet mode (files only) which is faster
	// Then parse to get package ownership
	cmd := exec.Command("pacman", "-Ql")
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.files = make(map[string][]string)
	c.fileOwners = make(map[string]string)

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		// Format: "package /path/to/file"
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}

		pkgName := parts[0]
		filePath := strings.TrimSpace(parts[1])

		// Skip directories (end with /)
		if filePath == "" || strings.HasSuffix(filePath, "/") {
			continue
		}

		c.files[pkgName] = append(c.files[pkgName], filePath)
		c.fileOwners[filePath] = pkgName
	}

	c.lastUpdate = time.Now()
	return nil
}

// LoadPackage loads files for a specific package
func (c *PackageFileCache) LoadPackage(packageName string) error {
	files, err := GetPacmanFiles(packageName)
	if err != nil {
		return err
	}
	c.SetPackageFiles(packageName, files)
	return nil
}

// GetOrLoadPackageFiles returns cached files or loads them if not cached
func (c *PackageFileCache) GetOrLoadPackageFiles(packageName string) ([]string, error) {
	if files, valid := c.GetPackageFiles(packageName); valid {
		return files, nil
	}

	if err := c.LoadPackage(packageName); err != nil {
		return nil, err
	}

	files, _ := c.GetPackageFiles(packageName)
	return files, nil
}

// GetAllFileOwners returns all file ownership mappings
func (c *PackageFileCache) GetAllFileOwners() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]string, len(c.fileOwners))
	for k, v := range c.fileOwners {
		result[k] = v
	}
	return result
}

// CheckConflicts checks if any of the given files conflict with cached ownership
func (c *PackageFileCache) CheckConflicts(files []string, excludePackage string) []FileConflict {
	c.mu.RLock()
	defer c.mu.RUnlock()

	conflicts := make([]FileConflict, 0)
	for _, f := range files {
		if owner, exists := c.fileOwners[f]; exists {
			if owner != excludePackage {
				conflicts = append(conflicts, FileConflict{
					FilePath:     f,
					OwningPkg:    owner,
					ConflictType: ConflictOwned,
				})
			}
		}
	}
	return conflicts
}

// FileConflict represents a file ownership conflict
type FileConflict struct {
	FilePath     string
	OwningPkg    string
	ConflictType FileConflictType
}

// FileConflictType indicates the type of file conflict
type FileConflictType int

const (
	ConflictOwned    FileConflictType = iota // File owned by another package
	ConflictOrphan                           // Unowned file exists
	ConflictModified                         // File modified from package version
)

// String returns a string representation
func (fct FileConflictType) String() string {
	switch fct {
	case ConflictOwned:
		return "owned"
	case ConflictOrphan:
		return "orphan"
	case ConflictModified:
		return "modified"
	default:
		return "unknown"
	}
}

// Global default cache instance
var defaultCache *PackageFileCache
var cacheOnce sync.Once

// GetDefaultCache returns the global package file cache
func GetDefaultCache() *PackageFileCache {
	cacheOnce.Do(func() {
		defaultCache = DefaultPackageFileCache()
	})
	return defaultCache
}

// GetInstalledPackageFilesCached is an optimized version using cache
func GetInstalledPackageFilesCached() (map[string][]string, error) {
	cache := GetDefaultCache()

	if !cache.IsValid() {
		if err := cache.LoadAll(); err != nil {
			return nil, err
		}
	}

	cache.mu.RLock()
	defer cache.mu.RUnlock()

	result := make(map[string][]string, len(cache.files))
	for k, v := range cache.files {
		result[k] = v
	}
	return result, nil
}
