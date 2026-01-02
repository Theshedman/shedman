package backends

import (
	"fmt"
	"net/http"
	"time"

	"github.com/theshedman/shedman/pkg/shedman/cache"
)

const (
	AURBaseURL    = "https://aur.archlinux.org/rpc/"
	AURAPIVersion = "5"
	HTTPTimeout   = 30 * time.Second
)

type AURBackend struct {
	baseURL      string
	client       *http.Client
	cache        cache.Cache
	forceRefresh bool // No-op for AUR (on-demand API)
}

// NewAURBackend creates a new AURBackend with default settings
func NewAURBackend(c cache.Cache) *AURBackend {
	return NewAURBackendWithURL(AURBaseURL, c)
}

// NewAURBackendWithURL creates a new AURBackend with a custom base URL (for testing)
func NewAURBackendWithURL(baseURL string, c cache.Cache) *AURBackend {
	return &AURBackend{
		baseURL: baseURL,
		client:  &http.Client{Timeout: HTTPTimeout},
		cache:   c,
	}
}

func (a *AURBackend) Name() string {
	return "aur"
}

// SetForceRefresh is a no-op for AUR since it's an on-demand API with no local cache
func (a *AURBackend) SetForceRefresh(force bool) {
	a.forceRefresh = force
}

// Sync verifies the AUR API is reachable.
// Unlike traditional repositories, the AUR is queried on-demand via its RPC API.
// There is no package database to download - packages are searched and fetched
// in real-time when needed (during install/search operations).
func (a *AURBackend) Sync() error {
	// Verify AUR API connectivity with a simple request
	url := fmt.Sprintf("%s?v=%s&type=info&arg=linux", a.baseURL, AURAPIVersion)

	resp, err := a.client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to reach AUR API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("AUR API returned status %d", resp.StatusCode)
	}

	// AUR is reachable and responding correctly
	return nil
}
