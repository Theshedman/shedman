package cache

import (
	"os"
	"path/filepath"
	"time"
)

const (
	DirPermission  = 0755 // rwxr-xr-x
	FilePermission = 0644 // rw-r--r--
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
	return os.MkdirAll(path, DirPermission)
}

func (c *FileSystemCache) WriteFile(path string, data []byte) error {
	if err := c.EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}
	return os.WriteFile(path, data, FilePermission)
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
