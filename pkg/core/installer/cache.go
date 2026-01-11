package installer

import (
	"strings"
	"sync"
	"time"

	"github.com/theshedman/shedman/pkg/backend"
	pacmanBackend "github.com/theshedman/shedman/pkg/backend/pacman"
)

// PackageFileCache provides thread-safe caching of package file ownership
type PackageFileCache struct {
	mu         sync.RWMutex
	files      map[string][]string // package name -> file list
	fileOwners map[string]string   // file path -> package name
	lastUpdate time.Time
	ttl        time.Duration
	backend    backend.OfficialBackend // Backend for file queries
}

// NewPackageFileCache creates a new cache with the specified TTL
// This auto-detects the backend; use NewPackageFileCacheWithBackend for explicit injection
func NewPackageFileCache(ttl time.Duration) *PackageFileCache {
	// Auto-detect backend
	var b backend.OfficialBackend
	if pacmanBackend.IsAlpmAvailable() {
		b, _ = pacmanBackend.New()
	}
	return NewPackageFileCacheWithBackend(ttl, b)
}

// NewPackageFileCacheWithBackend creates a cache with explicit backend injection
// This is the preferred constructor for production use and testing
func NewPackageFileCacheWithBackend(ttl time.Duration, b backend.OfficialBackend) *PackageFileCache {
	return &PackageFileCache{
		files:      make(map[string][]string),
		fileOwners: make(map[string]string),
		ttl:        ttl,
		backend:    b,
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

// LoadAll loads all installed package files using the backend
func (c *PackageFileCache) LoadAll() error {
	if c.backend == nil {
		return backend.ErrBackendNotFound
	}

	// Get all installed packages
	packages, err := c.backend.GetInstalledPackages()
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.files = make(map[string][]string)
	c.fileOwners = make(map[string]string)

	// Load files for each package
	for _, pkg := range packages {
		files, err := c.backend.GetPackageFiles(pkg.Name)
		if err != nil {
			continue // Skip packages with errors
		}

		for _, filePath := range files {
			// Skip directories (end with /)
			if filePath == "" || strings.HasSuffix(filePath, "/") {
				continue
			}

			c.files[pkg.Name] = append(c.files[pkg.Name], filePath)
			c.fileOwners[filePath] = pkg.Name
		}
	}

	c.lastUpdate = time.Now()
	return nil
}

// LoadPackage loads files for a specific package using the backend
func (c *PackageFileCache) LoadPackage(packageName string) error {
	if c.backend == nil {
		return backend.ErrBackendNotFound
	}

	files, err := c.backend.GetPackageFiles(packageName)
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
