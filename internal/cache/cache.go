package cache

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/theshedman/shedman/internal/util"
)

// Cache interface for file system operations
type Cache interface {
	GetDir() string
	GetSubDir(name string) string
	GetFilePath(subdir, filename string) string
	EnsureDir(path string) error
	WriteFile(path string, data []byte) error
	ReadFile(path string) ([]byte, error)
	IsFresh(path string, maxAge time.Duration) bool
	GetModTime(path string) (time.Time, error)
}

// FileSystemCache implements Cache using the real file system
type FileSystemCache struct {
	baseDir string
}

// NewFileSystemCache creates a cache using ~/.cache/shedman
func NewFileSystemCache() *FileSystemCache {
	cacheHome := os.Getenv("XDG_CACHE_HOME")
	if cacheHome == "" {
		cacheHome = constructDefaultCacheHome()
	}
	return &FileSystemCache{
		baseDir: filepath.Join(cacheHome, "shedman"),
	}
}

func constructDefaultCacheHome() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache")
}

// NewFileSystemCacheWithDir creates a cache with a custom base directory (for testing)
func NewFileSystemCacheWithDir(baseDir string) *FileSystemCache {
	return &FileSystemCache{baseDir: baseDir}
}

func (c *FileSystemCache) GetDir() string {
	return c.baseDir
}

func (c *FileSystemCache) GetSubDir(name string) string {
	return filepath.Join(c.baseDir, name)
}

func (c *FileSystemCache) GetFilePath(subdir, filename string) string {
	return filepath.Join(c.baseDir, subdir, filename)
}

func (c *FileSystemCache) EnsureDir(path string) error {
	return os.MkdirAll(path, util.DirPermissions)
}

func (c *FileSystemCache) WriteFile(path string, data []byte) error {
	if err := c.EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}
	return os.WriteFile(path, data, util.FilePermissions)
}

func (c *FileSystemCache) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// IsFresh checks if a cached file is younger than maxAge
func (c *FileSystemCache) IsFresh(path string, maxAge time.Duration) bool {
	modTime, err := c.GetModTime(path)
	if err != nil {
		return false
	}
	return time.Since(modTime) < maxAge
}

// GetModTime returns the modification time of a cached file
func (c *FileSystemCache) GetModTime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// CachedPackage represents a package file found in the cache
type CachedPackage struct {
	Name    string
	Version string
	Path    string
	ModTime time.Time
}

// FindVersions scans a directory for package files matching the given package name.
// It returns a list of matching packages.
func (c *FileSystemCache) FindVersions(dir string, pkgName string) ([]CachedPackage, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var matches []CachedPackage
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Simple extension check
		ext := filepath.Ext(name)
		if ext != ".zst" && ext != ".xz" && ext != ".shed" {
			continue
		}

		// Parse filename
		// Logic: {pkgname}-{ver}-{rel}-{arch}.{ext}
		// 1. Remove known extensions
		base := name
		if strings.HasSuffix(base, ".pkg.tar.zst") {
			base = strings.TrimSuffix(base, ".pkg.tar.zst")
		} else if strings.HasSuffix(base, ".pkg.tar.xz") {
			base = strings.TrimSuffix(base, ".pkg.tar.xz")
		} else if strings.HasSuffix(base, ".shed") {
			base = strings.TrimSuffix(base, ".shed")
		} else {
			// Fallback: strip last extension only if it's a known compression format
			// to avoid stripping version numbers like .1
			ext := filepath.Ext(base)
			if ext == ".zst" || ext == ".xz" || ext == ".gz" {
				base = strings.TrimSuffix(base, ext)
			}
		}

		// 2. Split by '-' from right
		parts := strings.Split(base, "-")
		if len(parts) < 4 {
			continue // Invalid format
		}

		// arch := parts[len(parts)-1]
		// rel := parts[len(parts)-2]
		ver := parts[len(parts)-3]
		pName := strings.Join(parts[:len(parts)-3], "-")

		if pName != pkgName {
			continue
		}

		fullVer := ver + "-" + parts[len(parts)-2] // ver-rel

		info, err := entry.Info()
		if err != nil {
			continue
		}

		matches = append(matches, CachedPackage{
			Name:    pName,
			Version: fullVer,
			Path:    filepath.Join(dir, name),
			ModTime: info.ModTime(),
		})
	}

	return matches, nil
}
