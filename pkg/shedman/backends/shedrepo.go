package backends

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

// ShedOS mirror list - primary and fallbacks
var DefaultMirrors = []string{
	"https://repo.shedos.org",
	"https://mirror1.shedos.org",
	"https://mirror2.shedos.org",
}

type ShedRepoBackend struct {
	client       *shedhttp.RetryClient
	cache        cache.Cache
	forceRefresh bool
}

func NewShedRepoBackend(c cache.Cache) *ShedRepoBackend {
	return NewShedRepoBackendWithMirrors(DefaultMirrors, c)
}

func NewShedRepoBackendWithURL(baseURL string, c cache.Cache) *ShedRepoBackend {
	return NewShedRepoBackendWithMirrors([]string{baseURL}, c)
}

func NewShedRepoBackendWithMirrors(mirrors []string, c cache.Cache) *ShedRepoBackend {
	return &ShedRepoBackend{
		client:       shedhttp.NewRetryClient(mirrors),
		cache:        c,
		forceRefresh: false,
	}
}

func (s *ShedRepoBackend) Name() string {
	return "shedrepo"
}

// SetForceRefresh sets whether to force re-download regardless of cache freshness
func (s *ShedRepoBackend) SetForceRefresh(force bool) {
	s.forceRefresh = force
}

func (s *ShedRepoBackend) Sync() error {
	// Sync arch database
	if err := s.syncAndCache(ArchDBPath, "shedos.db"); err != nil {
		return fmt.Errorf("failed to sync arch database: %w", err)
	}

	// Sync shed index
	if err := s.syncAndCache(ShedIndexPath, "index.json"); err != nil {
		return fmt.Errorf("failed to sync shed index: %w", err)
	}

	return nil
}

func (s *ShedRepoBackend) syncAndCache(urlPath, filename string) error {
	cacheFile := s.cache.GetFilePath("shedrepo", filename)

	// Check if cache is fresh (skip download if not forcing refresh)
	if !s.forceRefresh && s.cache.IsFresh(cacheFile, CacheMaxAge) {
		return nil
	}

	resp, err := s.client.Get(urlPath)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("endpoint %s returned status %d", urlPath, resp.StatusCode)
	}

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Save to cache
	if err := s.cache.WriteFile(cacheFile, body); err != nil {
		return fmt.Errorf("failed to cache %s: %w", filename, err)
	}

	return nil
}
