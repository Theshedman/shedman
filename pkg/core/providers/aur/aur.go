// Package aur implements the AUR (Arch User Repository) backend.
package aur

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strconv"
	"time"

	"github.com/theshedman/shedman/pkg/core"
)

const (
	AURBaseURL    = "https://aur.archlinux.org/rpc/"
	AURAPIVersion = "5"
	HTTPTimeout   = 30 * time.Second
)

// AURResponse represents the JSON response from AUR RPC API
type AURResponse struct {
	Version     int          `json:"version"`
	Type        string       `json:"type"`
	ResultCount int          `json:"resultcount"`
	Results     []AURPackage `json:"results"`
	Error       string       `json:"error,omitempty"`
}

// AURPackage represents a package from AUR RPC API
type AURPackage struct {
	ID             int     `json:"ID"`
	Name           string  `json:"Name"`
	PackageBaseID  int     `json:"PackageBaseID"`
	PackageBase    string  `json:"PackageBase"`
	Version        string  `json:"Version"`
	Description    string  `json:"Description"`
	URL            string  `json:"URL"`
	NumVotes       int     `json:"NumVotes"`
	Popularity     float64 `json:"Popularity"`
	OutOfDate      *int64  `json:"OutOfDate"`
	Maintainer     string  `json:"Maintainer"`
	FirstSubmitted int64   `json:"FirstSubmitted"`
	LastModified   int64   `json:"LastModified"`
	URLPath        string  `json:"URLPath"`
}

// Backend provides access to the Arch User Repository.
// AUR is queried on-demand via its RPC API - there is no local database.
type Backend struct {
	baseURL      string
	client       *http.Client
	cache        *core.PackageFileCache
	forceRefresh bool
}

// New creates a new AUR backend with default settings
func New(c *core.PackageFileCache) *Backend {
	return NewWithURL(AURBaseURL, c)
}

// NewWithURL creates a new AUR backend with a custom base URL (for testing)
func NewWithURL(baseURL string, c *core.PackageFileCache) *Backend {
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

// doRequest performs an HTTP GET request with proper headers
// falls back to curl if standard client fails (likely due to TLS fingerprinting/Cloudflare)
func (b *Backend) doRequest(url string) (*http.Response, error) {
	// Prefer curl if available to bypass TLS fingerprinting
	if hasCurl() {
		return b.curlRequest(url)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	return b.client.Do(req)
}

func hasCurl() bool {
	_, err := exec.LookPath("curl")
	return err == nil
}

func (b *Backend) curlRequest(url string) (*http.Response, error) {
	// Use -w %{http_code} to get the status code
	cmd := exec.Command("curl", "-s", "-L", "-A", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36", "-w", "%{http_code}", url)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	if len(output) < 3 {
		return nil, fmt.Errorf("invalid curl output")
	}

	statusStr := string(output[len(output)-3:])
	statusCode, err := strconv.Atoi(statusStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse status code: %w", err)
	}

	body := output[:len(output)-3]

	resp := &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewReader(body)),
		Header:     make(http.Header),
	}
	resp.Header.Set("Content-Type", "application/json")
	return resp, nil
}

// Sync verifies the AUR API is reachable.
// Unlike traditional repositories, the AUR is queried on-demand via its RPC API.
// There is no package database to download - packages are searched and fetched
// in real-time when needed (during install/search operations).
func (b *Backend) Sync() error {
	// Verify AUR API connectivity with a simple request
	reqURL := fmt.Sprintf("%s?v=%s&type=info&arg=linux", b.baseURL, AURAPIVersion)

	resp, err := b.doRequest(reqURL)
	if err != nil {
		return fmt.Errorf("failed to reach AUR API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("AUR API returned status %d", resp.StatusCode)
	}

	return nil
}

// Search searches AUR for packages matching the query
func (b *Backend) Search(query string) ([]core.PackageInfo, error) {
	// Build search URL
	reqURL := fmt.Sprintf("%s?v=%s&type=search&arg=%s",
		b.baseURL, AURAPIVersion, url.QueryEscape(query))

	resp, err := b.doRequest(reqURL)
	if err != nil {
		return nil, fmt.Errorf("AUR search failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AUR API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read AUR response: %w", err)
	}

	var aurResp AURResponse
	if err := json.Unmarshal(body, &aurResp); err != nil {
		if len(body) > 0 && body[0] == '<' {
			return nil, fmt.Errorf("received HTML response from AUR (possible API error or rate limit)")
		}
		return nil, fmt.Errorf("failed to parse AUR response: %w", err)
	}

	if aurResp.Error != "" {
		return nil, fmt.Errorf("AUR error: %s", aurResp.Error)
	}

	// Convert AUR packages to PackageInfo
	results := make([]core.PackageInfo, 0, len(aurResp.Results))
	for _, pkg := range aurResp.Results {
		results = append(results, core.PackageInfo{
			Name:        pkg.Name,
			Version:     pkg.Version,
			Description: pkg.Description,
			Source:      core.SourceAUR,
		})
	}

	return results, nil
}

// Info gets detailed information about a specific AUR package
func (b *Backend) Info(pkgName string) (*core.PackageInfo, error) {
	reqURL := fmt.Sprintf("%s?v=%s&type=info&arg=%s",
		b.baseURL, AURAPIVersion, url.QueryEscape(pkgName))

	resp, err := b.doRequest(reqURL)
	if err != nil {
		return nil, fmt.Errorf("AUR info failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AUR API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read AUR response: %w", err)
	}

	var aurResp AURResponse
	if err := json.Unmarshal(body, &aurResp); err != nil {
		return nil, fmt.Errorf("failed to parse AUR response: %w", err)
	}

	if aurResp.Error != "" {
		return nil, fmt.Errorf("AUR error: %s", aurResp.Error)
	}

	if len(aurResp.Results) == 0 {
		return nil, fmt.Errorf("package not found in AUR: %s", pkgName)
	}

	pkg := aurResp.Results[0]
	return &core.PackageInfo{
		Name:        pkg.Name,
		Version:     pkg.Version,
		Description: pkg.Description,
		Source:      core.SourceAUR,
	}, nil
}
