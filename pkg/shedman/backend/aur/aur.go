// Package aur implements the AUR (Arch User Repository) backend.
package aur

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

// Backend provides access to the Arch User Repository.
// AUR is queried on-demand via its RPC API - there is no local database.
type Backend struct {
	baseURL      string
	client       *http.Client
	cache        cache.Cache
	forceRefresh bool
}

// New creates a new AUR backend with default settings
func New(c cache.Cache) *Backend {
	return NewWithURL(AURBaseURL, c)
}

// NewWithURL creates a new AUR backend with a custom base URL (for testing)
func NewWithURL(baseURL string, c cache.Cache) *Backend {
	return &Backend{
		baseURL: baseURL,
		client:  &http.Client{Timeout: HTTPTimeout},
		cache:   c,
	}
}

// Name returns "aur"
func (b *Backend) Name() string {
	return "aur"
}

// SetForceRefresh is a no-op for AUR since it's an on-demand API
func (b *Backend) SetForceRefresh(force bool) {
	b.forceRefresh = force
}

// Sync verifies the AUR API is reachable.
// Unlike traditional repositories, the AUR is queried on-demand via its RPC API.
// There is no package database to download - packages are searched and fetched
// in real-time when needed (during install/search operations).
func (b *Backend) Sync() error {
	// Verify AUR API connectivity with a simple request
	url := fmt.Sprintf("%s?v=%s&type=info&arg=linux", b.baseURL, AURAPIVersion)

	resp, err := b.client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to reach AUR API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("AUR API returned status %d", resp.StatusCode)
	}

	return nil
}
