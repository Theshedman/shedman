package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/theshedman/shedman/internal/config"
)

// ShedOS API version path
const shedAPIPath = "/api/v1"

// Error types for ShedOS operations
var (
	ErrShedRequestFailed = errors.New("ShedOS repository request failed")
	ErrShedParseError    = errors.New("failed to parse ShedOS response")
)

// ShedPackage represents a package from the ShedOS repository
type ShedPackage struct {
	Name          string   `json:"name"`
	Version       string   `json:"version"`
	Description   string   `json:"description"`
	PackageType   string   `json:"package_type"` // "arch" or "shed"
	Depends       []string `json:"depends"`
	OptDepends    []string `json:"optdepends"`
	Provides      []string `json:"provides"`
	Conflicts     []string `json:"conflicts"`
	DownloadSize  int64    `json:"download_size"`
	InstalledSize int64    `json:"installed_size"`
	DownloadURL   string   `json:"download_url"` // Direct download URL
	Signature     string   `json:"signature"`
	Checksum      string   `json:"checksum"`
	Maintainer    string   `json:"maintainer"`
	URL           string   `json:"url"`
	License       string   `json:"license"`
}

// ShedSearchResponse represents the ShedOS search API response
type ShedSearchResponse struct {
	Packages []ShedPackage `json:"packages"`
	Total    int           `json:"total"`
	Error    string        `json:"error"`
}

// ShedInfoResponse represents the ShedOS package info API response
type ShedInfoResponse struct {
	Package *ShedPackage `json:"package"`
	Error   string       `json:"error"`
}

// ShedDB queries the ShedOS package repository
type ShedDB struct {
	httpClient HTTPClient
	baseURL    string
}

// NewShedDB creates a new ShedDB with default config
func NewShedDB() *ShedDB {
	cfg := config.Default()
	return NewShedDBWithConfig(cfg)
}

// NewShedDBWithConfig creates a new ShedDB with the given config
func NewShedDBWithConfig(cfg *config.Config) *ShedDB {
	baseURL := ""
	if len(cfg.Mirrors.ShedOS) > 0 {
		baseURL = cfg.Mirrors.ShedOS[0]
	}
	if baseURL == "" {
		baseURL = "https://repo.shedos.org"
	}

	return &ShedDB{
		httpClient: DefaultHTTPClient,
		baseURL:    baseURL + shedAPIPath,
	}
}

// SetHTTPClient sets a custom HTTP client (for testing)
func (s *ShedDB) SetHTTPClient(client HTTPClient) {
	s.httpClient = client
}

// Search searches for packages matching the query
func (s *ShedDB) Search(query string) ([]PackageInfo, error) {
	searchURL := s.buildSearchURL(query)
	data, err := s.httpClient(searchURL)
	if err != nil {
		return nil, err
	}

	var response ShedSearchResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrShedParseError, err)
	}

	if response.Error != "" {
		return nil, fmt.Errorf("%w: %s", ErrShedRequestFailed, response.Error)
	}

	results := make([]PackageInfo, 0, len(response.Packages))
	for _, pkg := range response.Packages {
		results = append(results, ShedPackageToPackageInfo(pkg))
	}

	return results, nil
}

// GetInfo returns detailed info about a package
func (s *ShedDB) GetInfo(name string) (*PackageInfo, error) {
	infoURL := s.buildInfoURL(name)
	data, err := s.httpClient(infoURL)
	if err != nil {
		return nil, err
	}

	var response ShedInfoResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrShedParseError, err)
	}

	if response.Error != "" || response.Package == nil {
		return nil, nil
	}

	info := ShedPackageToPackageInfo(*response.Package)
	return &info, nil
}

// buildSearchURL constructs the ShedOS search URL
func (s *ShedDB) buildSearchURL(query string) string {
	return fmt.Sprintf("%s/search?q=%s", s.baseURL, url.QueryEscape(query))
}

// buildInfoURL constructs the ShedOS package info URL
func (s *ShedDB) buildInfoURL(name string) string {
	return fmt.Sprintf("%s/package/%s", s.baseURL, url.PathEscape(name))
}

// BuildSearchURL returns the search URL for testing visibility
func (s *ShedDB) BuildSearchURL(query string) string {
	return s.buildSearchURL(query)
}

// BuildInfoURL returns the info URL for testing visibility
func (s *ShedDB) BuildInfoURL(name string) string {
	return s.buildInfoURL(name)
}

// ShedPackageToPackageInfo converts a ShedOS package to PackageInfo
func ShedPackageToPackageInfo(pkg ShedPackage) PackageInfo {
	pkgType := pkg.PackageType
	if pkgType == "" {
		pkgType = PackageTypeArch // Default to Arch format
	}

	return PackageInfo{
		Name:          pkg.Name,
		Version:       pkg.Version,
		Description:   pkg.Description,
		Source:        SourceShedOS,
		PackageType:   pkgType,
		Depends:       pkg.Depends,
		OptDepends:    pkg.OptDepends,
		Provides:      pkg.Provides,
		Conflicts:     pkg.Conflicts,
		Size:          pkg.DownloadSize,
		InstalledSize: pkg.InstalledSize,
		DownloadURL:   pkg.DownloadURL,
		Checksum:      pkg.Checksum,
		Signature:     pkg.Signature,
	}
}
