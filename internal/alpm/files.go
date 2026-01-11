// Package pacman provides file ownership queries for pacman.
package alpm

import (
	"bufio"
	"strings"
	"sync"
	"time"
)

// FileCache provides thread-safe caching of package file ownership
type FileCache struct {
	mu         sync.RWMutex
	files      map[string][]string // package name -> file list
	fileOwners map[string]string   // file path -> package name
	lastUpdate time.Time
	ttl        time.Duration
	executor   CommandExecutor
}

// NewFileCache creates a new cache with the specified TTL
func NewFileCache(ttl time.Duration) *FileCache {
	return &FileCache{
		files:      make(map[string][]string),
		fileOwners: make(map[string]string),
		ttl:        ttl,
		executor:   &RealExecutor{},
	}
}

// DefaultFileCache creates a cache with 5-minute TTL
func DefaultFileCache() *FileCache {
	return NewFileCache(5 * time.Minute)
}

// IsValid checks if the cache is still valid
func (c *FileCache) IsValid() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.lastUpdate.IsZero() {
		return false
	}
	return time.Since(c.lastUpdate) < c.ttl
}

// Invalidate clears the cache
func (c *FileCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.files = make(map[string][]string)
	c.fileOwners = make(map[string]string)
	c.lastUpdate = time.Time{}
}

// GetPackageFiles returns files for a package using pacman -Ql
func GetPackageFiles(pkgName string) ([]string, error) {
	executor := &RealExecutor{}
	output, err := executor.Output("pacman", "-Ql", pkgName)
	if err != nil {
		return nil, err
	}

	var files []string
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			filePath := strings.TrimSpace(parts[1])
			if filePath != "" && !strings.HasSuffix(filePath, "/") {
				files = append(files, filePath)
			}
		}
	}

	return files, nil
}

// GetPackageFilesFromCache returns files for a package, using cache if valid
func (c *FileCache) GetPackageFilesFromCache(packageName string) ([]string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	files, exists := c.files[packageName]
	return files, exists && c.IsValid()
}

// GetFileOwner returns the package that owns a file
func (c *FileCache) GetFileOwner(filePath string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	owner, exists := c.fileOwners[filePath]
	return owner, exists && c.IsValid()
}

// SetPackageFiles stores files for a package
func (c *FileCache) SetPackageFiles(packageName string, files []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.files[packageName] = files
	for _, f := range files {
		c.fileOwners[f] = packageName
	}
	c.lastUpdate = time.Now()
}

// LoadAll loads all installed package files using batch query
func (c *FileCache) LoadAll() error {
	output, err := c.executor.Output("pacman", "-Ql")
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
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}

		pkgName := parts[0]
		filePath := strings.TrimSpace(parts[1])

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
func (c *FileCache) LoadPackage(packageName string) error {
	files, err := GetPackageFiles(packageName)
	if err != nil {
		return err
	}
	c.SetPackageFiles(packageName, files)
	return nil
}

// GetOrLoadPackageFiles returns cached files or loads them if not cached
func (c *FileCache) GetOrLoadPackageFiles(packageName string) ([]string, error) {
	if files, valid := c.GetPackageFilesFromCache(packageName); valid {
		return files, nil
	}

	if err := c.LoadPackage(packageName); err != nil {
		return nil, err
	}

	files, _ := c.GetPackageFilesFromCache(packageName)
	return files, nil
}

// GetAllFileOwners returns all file ownership mappings
func (c *FileCache) GetAllFileOwners() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]string, len(c.fileOwners))
	for k, v := range c.fileOwners {
		result[k] = v
	}
	return result
}

// Global default cache instance
var defaultCache *FileCache
var cacheOnce sync.Once

// GetDefaultCache returns the global package file cache
func GetDefaultCache() *FileCache {
	cacheOnce.Do(func() {
		defaultCache = DefaultFileCache()
	})
	return defaultCache
}
