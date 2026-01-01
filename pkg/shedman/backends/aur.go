package backends

import (
"encoding/json"
"fmt"
"io"
"net/http"
"time"

"github.com/theshedman/shedman/pkg/shedman/cache"
)

const (
AURBaseURL    = "https://aur.archlinux.org/rpc/"
AURAPIVersion = "5"
HTTPTimeout   = 30 * time.Second
)

// AURResponse represents the AUR RPC API response structure
type AURResponse struct {
	Version     int           `json:"version"`
	Type        string        `json:"type"`
	ResultCount int           `json:"resultcount"`
	Results     []interface{} `json:"results"`
	Error       string        `json:"error,omitempty"`
}

type AURBackend struct {
	baseURL string
	client  *http.Client
	cache   cache.Cache
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

func (a *AURBackend) Sync() error {
	url := fmt.Sprintf("%s?v=%s&type=info&arg=linux", a.baseURL, AURAPIVersion)

	resp, err := a.client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to reach AUR API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("AUR API returned status %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read AUR response: %w", err)
	}

	// Parse to check for API errors
	var result AURResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to decode AUR response: %w", err)
	}

	if result.Error != "" {
		return fmt.Errorf("AUR API error: %s", result.Error)
	}

	// Save to cache
	cacheFile := a.cache.GetFilePath("aur", "packages.json")
	if err := a.cache.WriteFile(cacheFile, body); err != nil {
		return fmt.Errorf("failed to cache AUR data: %w", err)
	}

	return nil
}
