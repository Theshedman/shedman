package backends

import (
	"fmt"
	"net/http"
)

const (
	ShedRepoBaseURL = "https://repo.shedos.org"
	ArchDBPath      = "/arch/x86_64/shedos.db"
	ShedIndexPath   = "/shed/index.json"
)

type ShedRepoBackend struct {
	baseURL string
	client  *http.Client
}

func NewShedRepoBackend() *ShedRepoBackend {
	return NewShedRepoBackendWithURL(ShedRepoBaseURL)
}

func NewShedRepoBackendWithURL(baseURL string) *ShedRepoBackend {
	return &ShedRepoBackend{
		baseURL: baseURL,
		client:  &http.Client{Timeout: HTTPTimeout},
	}
}
	
func (s *ShedRepoBackend) Name() string {
	return "shedrepo"
}

func (s *ShedRepoBackend) Sync() error {
	// Sync arch database
	if err := s.syncEndpoint(ArchDBPath); err != nil {
		return fmt.Errorf("failed to sync arch database: %w", err)
	}
	// Sync shed index
	if err := s.syncEndpoint(ShedIndexPath); err != nil {
		return fmt.Errorf("failed to sync shed index: %w", err)
	}
	return nil
}

func (s *ShedRepoBackend) syncEndpoint(path string) error {
	resp, err := s.client.Get(s.baseURL + path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("endpoint %s returned status %d", path, resp.StatusCode)
	}
	return nil
}