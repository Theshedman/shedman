// Package shedrepo implements the ShedOS repository backend.
package shedrepo

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/theshedman/shedman/pkg/shedman/cache"
	shedhttp "github.com/theshedman/shedman/pkg/shedman/http"
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

// Backend provides access to the ShedOS repository.
type Backend struct {
	client       *shedhttp.RetryClient
	cache        cache.Cache
	forceRefresh bool
}

// New creates a new ShedRepo backend with default settings
func New(c cache.Cache) *Backend {
	return NewWithMirrors(DefaultMirrors, c)
}

// NewWithURL creates a new ShedRepo backend with a custom base URL
func NewWithURL(baseURL string, c cache.Cache) *Backend {
	return NewWithMirrors([]string{baseURL}, c)
}

// NewWithMirrors creates a new ShedRepo backend with custom mirrors
func NewWithMirrors(mirrors []string, c cache.Cache) *Backend {
	return &Backend{
		client:       shedhttp.NewRetryClient(mirrors),
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
	defer resp.Body.Close()

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
