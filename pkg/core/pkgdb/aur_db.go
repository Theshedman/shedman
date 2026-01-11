package pkgdb

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/theshedman/shedman/pkg/shedman/config"
)

// AUR RPC API version path
const aurRPCPath = "/rpc/v5"

// Error types for AUR operations
var (
	ErrAURRequestFailed = errors.New("AUR RPC request failed")
	ErrAURParseError    = errors.New("failed to parse AUR response")
)

// HTTPClient is a function that performs HTTP GET requests
type HTTPClient func(url string) ([]byte, error)

// DefaultHTTPClient performs real HTTP requests
func DefaultHTTPClient(reqURL string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAURRequestFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrAURRequestFailed, resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// AURPackage represents a package from the AUR RPC API
type AURPackage struct {
	Name        string   `json:"Name"`
	Version     string   `json:"Version"`
	Description string   `json:"Description"`
	Depends     []string `json:"Depends"`
	OptDepends  []string `json:"OptDepends"`
	Provides    []string `json:"Provides"`
	Conflicts   []string `json:"Conflicts"`
	NumVotes    int      `json:"NumVotes"`
	Popularity  float64  `json:"Popularity"`
	Maintainer  string   `json:"Maintainer"`
	URL         string   `json:"URL"`
	OutOfDate   *int64   `json:"OutOfDate"`
}

// AURSearchResponse represents the AUR RPC search response
type AURSearchResponse struct {
	ResultCount int          `json:"resultcount"`
	Results     []AURPackage `json:"results"`
	Type        string       `json:"type"`
	Error       string       `json:"error"`
}

// AURInfoResponse represents the AUR RPC info response
type AURInfoResponse struct {
	ResultCount int          `json:"resultcount"`
	Results     []AURPackage `json:"results"`
	Type        string       `json:"type"`
	Error       string       `json:"error"`
}

// AURDB queries the Arch User Repository via RPC API
type AURDB struct {
	httpClient HTTPClient
	baseURL    string
}

// NewAURDB creates a new AURDB with default config
func NewAURDB() *AURDB {
	cfg := config.Default()
	return NewAURDBWithConfig(cfg)
}

// NewAURDBWithConfig creates a new AURDB with the given config
func NewAURDBWithConfig(cfg *config.Config) *AURDB {
	baseURL := cfg.Mirrors.AUR
	if baseURL == "" {
		baseURL = "https://aur.archlinux.org"
	}

	return &AURDB{
		httpClient: DefaultHTTPClient,
		baseURL:    baseURL + aurRPCPath,
	}
}

// SetHTTPClient sets a custom HTTP client (for testing)
func (a *AURDB) SetHTTPClient(client HTTPClient) {
	a.httpClient = client
}

// Search searches for packages matching the query
func (a *AURDB) Search(query string) ([]PackageInfo, error) {
	searchURL := a.buildSearchURL(query)
	data, err := a.httpClient(searchURL)
	if err != nil {
		return nil, err
	}

	var response AURSearchResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAURParseError, err)
	}

	if response.Error != "" {
		return nil, fmt.Errorf("%w: %s", ErrAURRequestFailed, response.Error)
	}

	results := make([]PackageInfo, 0, len(response.Results))
	for _, pkg := range response.Results {
		results = append(results, AURPackageToPackageInfo(pkg))
	}

	return results, nil
}

// GetInfo returns detailed info about a package
func (a *AURDB) GetInfo(name string) (*PackageInfo, error) {
	infoURL := a.buildInfoURL(name)
	data, err := a.httpClient(infoURL)
	if err != nil {
		return nil, err
	}

	var response AURInfoResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAURParseError, err)
	}

	if response.Error != "" {
		return nil, fmt.Errorf("%w: %s", ErrAURRequestFailed, response.Error)
	}

	if response.ResultCount == 0 {
		return nil, nil
	}

	info := AURPackageToPackageInfo(response.Results[0])
	return &info, nil
}

// buildSearchURL constructs the AUR RPC search URL
func (a *AURDB) buildSearchURL(query string) string {
	return fmt.Sprintf("%s/search/%s", a.baseURL, url.PathEscape(query))
}

// buildInfoURL constructs the AUR RPC info URL
func (a *AURDB) buildInfoURL(name string) string {
	return fmt.Sprintf("%s/info?arg[]=%s", a.baseURL, url.QueryEscape(name))
}

// BuildSearchURL returns the search URL for testing visibility
func (a *AURDB) BuildSearchURL(query string) string {
	return a.buildSearchURL(query)
}

// BuildInfoURL returns the info URL for testing visibility
func (a *AURDB) BuildInfoURL(name string) string {
	return a.buildInfoURL(name)
}

// AURPackageToPackageInfo converts an AUR package to PackageInfo
func AURPackageToPackageInfo(pkg AURPackage) PackageInfo {
	return PackageInfo{
		Name:        pkg.Name,
		Version:     pkg.Version,
		Description: pkg.Description,
		Source:      SourceAUR,
		Depends:     pkg.Depends,
		OptDepends:  pkg.OptDepends,
		Provides:    pkg.Provides,
		Conflicts:   pkg.Conflicts,
	}
}
