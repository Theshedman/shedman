package backends

import (
"fmt"
"io"
"net/http"

"github.com/theshedman/shedman/pkg/shedman/cache"
)

const (
ShedRepoBaseURL = "https://repo.shedos.org"
ArchDBPath      = "/arch/x86_64/shedos.db"
ShedIndexPath   = "/shed/index.json"
)

type ShedRepoBackend struct {
	baseURL string
	client  *http.Client
	cache   cache.Cache
}

func NewShedRepoBackend(c cache.Cache) *ShedRepoBackend {
	return NewShedRepoBackendWithURL(ShedRepoBaseURL, c)
}

func NewShedRepoBackendWithURL(baseURL string, c cache.Cache) *ShedRepoBackend {
	return &ShedRepoBackend{
		baseURL: baseURL,
		client:  &http.Client{Timeout: HTTPTimeout},
		cache:   c,
	}
}

func (s *ShedRepoBackend) Name() string {
	return "shedrepo"
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
	resp, err := s.client.Get(s.baseURL + urlPath)
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
	cacheFile := s.cache.GetFilePath("shedrepo", filename)
	if err := s.cache.WriteFile(cacheFile, body); err != nil {
		return fmt.Errorf("failed to cache %s: %w", filename, err)
	}

	return nil
}
