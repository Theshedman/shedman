// Package shedrepo implements the ShedOS repository backend.
package shedrepo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	shedhttp "github.com/theshedman/shedman/internal/http"
	"github.com/theshedman/shedman/pkg/core"
)

const (
	ShedRepoBaseURL = "https://repo.shedos.org"
	ArchDBPath      = "/arch/x86_64/shedos.db"
	ShedIndexPath   = "/shed/index.json"
	CacheMaxAge     = 1 * time.Hour
)

// DefaultMirrors is the ShedOS mirror list
var DefaultMirrors = []string{
	"https://repo.shedos.org",
	"https://mirror1.shedos.org",
	"https://mirror2.shedos.org",
}

// ShedIndex represents the index.json structure
type ShedIndex struct {
	Version  int           `json:"version"`
	Updated  string        `json:"updated"`
	Packages []ShedPackage `json:"packages"`
}

// ShedPackage represents a package in the ShedOS index
type ShedPackage struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Depends     []string `json:"depends,omitempty"`
	URL         string   `json:"url,omitempty"`
	Size        int64    `json:"size,omitempty"`
}

// Backend provides access to the ShedOS repository.
type Backend struct {
	client       *shedhttp.RetryClient
	cache        *core.FileSystemCache
	forceRefresh bool
}

// New creates a new ShedRepo backend with default settings
func New(c *core.FileSystemCache, timeout time.Duration) *Backend {
	return NewWithMirrors(DefaultMirrors, c, timeout)
}

// NewWithURL creates a new ShedRepo backend with a custom base URL
func NewWithURL(baseURL string, c *core.FileSystemCache, timeout time.Duration) *Backend {
	return NewWithMirrors([]string{baseURL}, c, timeout)
}

// NewWithMirrors creates a new ShedRepo backend with custom mirrors
func NewWithMirrors(mirrors []string, c *core.FileSystemCache, timeout time.Duration) *Backend {
	return &Backend{
		client:       shedhttp.NewRetryClient(mirrors, timeout),
		cache:        c,
		forceRefresh: false,
	}
}

// Name returns "shedrepo"
func (b *Backend) Name() string {
	return "shedrepo"
}

// SetForceRefresh sets whether to force re-download regardless of cache freshness
func (b *Backend) SetForceRefresh(force bool) {
	b.forceRefresh = force
}

// Sync downloads/updates the ShedOS repository database and index
func (b *Backend) Sync() error {
	// Sync arch database
	if err := b.syncAndCache(ArchDBPath, "shedos.db"); err != nil {
		return fmt.Errorf("failed to sync arch database: %w", err)
	}

	// Sync shed index
	if err := b.syncAndCache(ShedIndexPath, "index.json"); err != nil {
		return fmt.Errorf("failed to sync shed index: %w", err)
	}

	return nil
}

func (b *Backend) syncAndCache(urlPath, filename string) error {
	cacheFile := b.cache.GetFilePath("shedrepo", filename)

	// Check if cache is fresh (skip download if not forcing refresh)
	if !b.forceRefresh && b.cache.IsFresh(cacheFile, CacheMaxAge) {
		return nil
	}

	resp, err := b.client.Get(urlPath)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("endpoint %s returned status %d", urlPath, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if err := b.cache.WriteFile(cacheFile, body); err != nil {
		return fmt.Errorf("failed to cache %s: %w", filename, err)
	}

	return nil
}

// loadIndex loads and parses the cached index.json
func (b *Backend) loadIndex() (*ShedIndex, error) {
	cacheFile := b.cache.GetFilePath("shedrepo", "index.json")

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Index not cached, try to sync first
			if err := b.Sync(); err != nil {
				return nil, fmt.Errorf("failed to sync: %w", err)
			}
			data, err = os.ReadFile(cacheFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read index after sync: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to read cached index: %w", err)
		}
	}

	var index ShedIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("failed to parse index: %w", err)
	}

	return &index, nil
}

// Search searches ShedOS repository for packages matching the query
func (b *Backend) Search(query string) ([]core.PackageInfo, error) {
	index, err := b.loadIndex()
	if err != nil {
		return nil, err
	}

	query = strings.ToLower(query)
	var results []core.PackageInfo

	for _, pkg := range index.Packages {
		if strings.Contains(strings.ToLower(pkg.Name), query) ||
			strings.Contains(strings.ToLower(pkg.Description), query) {
			results = append(results, core.PackageInfo{
				Name:        pkg.Name,
				Version:     pkg.Version,
				Description: pkg.Description,
				Source:      core.SourceShedOS,
			})
		}
	}

	return results, nil
}

// Info gets information about a specific ShedOS package
func (b *Backend) Info(pkgName string) (*core.PackageInfo, error) {
	index, err := b.loadIndex()
	if err != nil {
		return nil, err
	}

	for _, pkg := range index.Packages {
		if pkg.Name == pkgName {
			return &core.PackageInfo{
				Name:        pkg.Name,
				Version:     pkg.Version,
				Description: pkg.Description,
				Source:      core.SourceShedOS,
			}, nil
		}
	}

	return nil, fmt.Errorf("package not found in ShedOS repo: %s", pkgName)
}
