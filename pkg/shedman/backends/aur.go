package backends

import (
"encoding/json"
"fmt"
"net/http"
"time"
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
}

// NewAURBackend creates a new AURBackend with default settings
func NewAURBackend() *AURBackend {
	return NewAURBackendWithURL(AURBaseURL)
}

// NewAURBackendWithURL creates a new AURBackend with a custom base URL (for testing)
func NewAURBackendWithURL(baseURL string) *AURBackend {
	return &AURBackend{
		baseURL: baseURL,
		client:  &http.Client{Timeout: HTTPTimeout},
	}
}

func (a *AURBackend) Name() string {
	return "aur"
}

func (a *AURBackend) Sync() error {
	// Verify AUR API is reachable by making a simple info request
	url := fmt.Sprintf("%s?v=%s&type=info&arg=linux", a.baseURL, AURAPIVersion)

	resp, err := a.client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to reach AUR API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("AUR API returned status %d", resp.StatusCode)
	}

	var result AURResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode AUR response: %w", err)
	}

	if result.Error != "" {
		return fmt.Errorf("AUR API error: %s", result.Error)
	}

	return nil
}
