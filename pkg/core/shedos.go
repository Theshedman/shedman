package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/theshedman/shedman/internal/config"
)

// DownloadProgressCallback reports download progress
type DownloadProgressCallback func(downloaded, total int64, speed float64)

// ShedOSInstaller handles downloading and installing ShedOS packages
type ShedOSInstaller struct {
	executor          Executor
	cacheDir          string
	httpClient        *http.Client
	timeout           time.Duration
	retries           int
	parallelDownloads int // Default parallel downloads
	shed              *ShedInstaller
	backend           OfficialBackend // Backend for local package installation
}

// NewShedOSInstallerWithBackend creates a new ShedOSInstaller with explicit backend injection
// This is the preferred constructor for production use and testing
func NewShedOSInstallerWithBackend(cfg *config.Config, b OfficialBackend) *ShedOSInstaller {
	// Set default cache directory
	home, _ := os.UserHomeDir()
	cacheDir := filepath.Join(home, ".cache", "shedman", "shedos")

	// Read timeout from config (convert seconds to duration)
	timeout := time.Duration(cfg.Network.Timeout) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	// Read retries from config
	retries := cfg.Network.Retry
	if retries == 0 {
		retries = 3
	}

	parallel := cfg.Network.ParallelDownloads
	if parallel <= 0 {
		parallel = 5
	}

	return &ShedOSInstaller{
		executor:          DefaultExecutor,
		cacheDir:          cacheDir,
		httpClient:        &http.Client{Timeout: timeout},
		timeout:           timeout,
		retries:           retries,
		parallelDownloads: parallel,
		shed:              NewShedInstaller(),
		backend:           b,
	}
}

// SetCacheDir sets the download cache directory
func (s *ShedOSInstaller) SetCacheDir(dir string) {
	s.cacheDir = dir
}

// SetExecutor sets a custom executor for testing
func (s *ShedOSInstaller) SetExecutor(exec Executor) {
	s.executor = exec
}

// Download downloads a package from ShedOS repository
func (s *ShedOSInstaller) Download(pkg PackageInfo) (string, error) {
	return s.DownloadWithProgress(pkg, nil)
}

// DownloadWithProgress downloads a package with progress reporting
func (s *ShedOSInstaller) DownloadWithProgress(pkg PackageInfo, callback DownloadProgressCallback) (string, error) {
	if pkg.DownloadURL == "" {
		return "", fmt.Errorf("no download URL for package %s", pkg.Name)
	}

	// Create cache directory
	if err := os.MkdirAll(s.cacheDir, DirPermissions); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Determine filename
	filename := pkg.Name + "-" + pkg.Version
	if pkg.IsShedFormat() {
		filename += ".shed"
	} else {
		filename += ".pkg.tar.zst"
	}

	destPath := filepath.Join(s.cacheDir, filename)

	// Download with retries
	var lastErr error
	for attempt := 0; attempt < s.retries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * time.Second) // Exponential backoff
		}

		err := s.downloadFile(pkg.DownloadURL, destPath, pkg.Size, callback)
		if err == nil {
			return destPath, nil
		}
		lastErr = err
	}

	return "", fmt.Errorf("download failed after %d attempts: %w", s.retries, lastErr)
}

// DownloadResult contains paths for downloaded files
type DownloadResult struct {
	PkgPath string
	SigPath string
}

// DownloadMultiple downloads multiple packages in parallel
func (s *ShedOSInstaller) DownloadMultiple(pkgs []PackageInfo, callback DownloadProgressCallback) (map[string]DownloadResult, error) {
	if len(pkgs) == 0 {
		return nil, nil
	}

	results := make(map[string]DownloadResult)
	errors := make(chan error, len(pkgs))
	resultsChan := make(chan struct {
		name string
		res  DownloadResult
	}, len(pkgs))

	// Create semaphore to control concurrency
	concurrency := s.parallelDownloads
	if concurrency > len(pkgs) {
		concurrency = len(pkgs)
	}
	sem := make(chan struct{}, concurrency)

	var wg sync.WaitGroup

	for _, pkg := range pkgs {
		wg.Add(1)
		go func(p PackageInfo) {
			defer wg.Done()

			sem <- struct{}{}        // Acquire token
			defer func() { <-sem }() // Release token

			pkgPath, err := s.DownloadWithProgress(p, callback)
			if err != nil {
				errors <- fmt.Errorf("failed to download %s: %w", p.Name, err)
				return
			}

			// Download signature (best effort)
			sigPath, err := s.DownloadSignature(p)
			if err != nil {
				// Ignore error for signature download as it's optional/best-effort here
			}

			resultsChan <- struct {
				name string
				res  DownloadResult
			}{p.Name, DownloadResult{PkgPath: pkgPath, SigPath: sigPath}}
		}(pkg)
	}

	// Wait for all downloads to finish
	wg.Wait()
	close(errors)
	close(resultsChan)

	// Collect errors first
	var errMsgs []string
	for err := range errors {
		errMsgs = append(errMsgs, err.Error())
	}

	// Collect results
	for res := range resultsChan {
		results[res.name] = res.res
	}

	// Check if any errors occurred
	if len(errMsgs) > 0 {
		return nil, fmt.Errorf("multiple download errors: %s", strings.Join(errMsgs, "; "))
	}

	return results, nil
}

// downloadFile performs the actual file download with progress
func (s *ShedOSInstaller) downloadFile(url, destPath string, expectedSize int64, callback DownloadProgressCallback) error {
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("GET request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Get content length
	totalSize := resp.ContentLength
	if totalSize <= 0 {
		totalSize = expectedSize
	}

	// Create destination file
	file, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy with progress tracking
	var downloaded int64
	startTime := time.Now()
	buf := make([]byte, 32*1024) // 32KB buffer

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := file.Write(buf[:n]); writeErr != nil {
				return fmt.Errorf("failed to write: %w", writeErr)
			}
			downloaded += int64(n)

			if callback != nil {
				elapsed := time.Since(startTime).Seconds()
				if elapsed > 0 {
					speed := float64(downloaded) / elapsed / 1024 // KB/s
					callback(downloaded, totalSize, speed)
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read: %w", err)
		}
	}

	return nil
}

// DownloadSignature downloads the GPG signature for a package
func (s *ShedOSInstaller) DownloadSignature(pkg PackageInfo) (string, error) {
	// Determine signature URL
	sigURL := pkg.Signature
	if sigURL == "" {
		// Try default .sig extension
		sigURL = pkg.DownloadURL + ".sig"
	}

	// Check if it's a URL (not a local path)
	if !strings.HasPrefix(sigURL, "http://") && !strings.HasPrefix(sigURL, "https://") {
		// It's a local signature or doesn't exist
		return sigURL, nil
	}

	// Create signature destination path
	filename := pkg.Name + "-" + pkg.Version + ".sig"
	sigPath := filepath.Join(s.cacheDir, filename)

	// Download with retries
	var lastErr error
	for attempt := 0; attempt < s.retries; attempt++ {
		resp, err := s.httpClient.Get(sigURL)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode == http.StatusNotFound {
			resp.Body.Close()
			return "", nil // No signature available
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
			continue
		}

		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}

		if err := os.WriteFile(sigPath, data, FilePermissions); err != nil {
			return "", err
		}

		return sigPath, nil
	}

	return "", fmt.Errorf("signature download failed: %w", lastErr)
}

// VerifyChecksum verifies the SHA256 checksum of a downloaded file
func (s *ShedOSInstaller) VerifyChecksum(filePath, expectedChecksum string) error {
	if expectedChecksum == "" {
		return nil // No checksum to verify
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return err
	}

	actualChecksum := hex.EncodeToString(hasher.Sum(nil))
	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return nil
}

// VerifyGPGSignature verifies the GPG signature of a package
func (s *ShedOSInstaller) VerifyGPGSignature(filePath, sigPath string) error {
	if !CommandExists("gpg") {
		return ErrGPGNotFound
	}

	// Check if sig file exists
	if sigPath == "" {
		sigPath = filePath + ".sig"
	}

	if _, err := os.Stat(sigPath); os.IsNotExist(err) {
		return nil // No signature to verify
	}

	cmd := []string{"gpg", "--verify", sigPath, filePath}
	return s.executor("", cmd)
}

// Install installs a package from ShedOS repository
func (s *ShedOSInstaller) Install(pkg PackageInfo, opts Options) error {
	return s.InstallWithProgress(pkg, opts, nil)
}

// InstallWithProgress installs a package with progress callbacks
func (s *ShedOSInstaller) InstallWithProgress(pkg PackageInfo, opts Options, callback DownloadProgressCallback) error {
	// Download the package
	pkgPath, err := s.DownloadWithProgress(pkg, callback)
	if err != nil {
		return err
	}

	// Download signature
	sigPath, err := s.DownloadSignature(pkg)
	if err != nil {
		// Log but don't fail
		fmt.Fprintf(os.Stderr, "Warning: could not download signature: %v\n", err)
	}

	// Verify checksum
	if err := s.VerifyChecksum(pkgPath, pkg.Checksum); err != nil {
		return fmt.Errorf("checksum verification failed: %w", err)
	}

	// Verify GPG signature
	if sigPath != "" {
		if err := s.VerifyGPGSignature(pkgPath, sigPath); err != nil {
			return fmt.Errorf("GPG verification failed: %w", err)
		}
	}

	// Install based on package type
	if pkg.IsShedFormat() {
		return s.shed.Install(pkgPath)
	}

	// Native Arch package - use pacman -U
	return s.InstallWithPacman(pkgPath, opts)
}

// InstallWithPacman installs a local package file using the backend
func (s *ShedOSInstaller) InstallWithPacman(pkgPath string, opts Options) error {
	// Backend is required - no fallback to pacman binary
	if s.backend == nil {
		return fmt.Errorf("no backend available for package installation")
	}
	return fmt.Errorf("local shedOS package installation not yet implemented")
}

// InstallMultiple installs multiple packages from ShedOS
func (s *ShedOSInstaller) InstallMultiple(pkgs []PackageInfo, opts Options) error {
	// 1. Download all packages in parallel first
	downloadedMap, err := s.DownloadMultiple(pkgs, nil)
	if err != nil {
		return err
	}

	// 2. Install each package using the downloaded files
	for _, pkg := range pkgs {
		result, ok := downloadedMap[pkg.Name]
		if !ok {
			return fmt.Errorf("download result not found for %s", pkg.Name)
		}
		pkgPath := result.PkgPath
		sigPath := result.SigPath

		// Verify Checksum
		if err := s.VerifyChecksum(pkgPath, pkg.Checksum); err != nil {
			return fmt.Errorf("checksum verification failed for %s: %w", pkg.Name, err)
		}

		// Verify GPG
		if sigPath != "" {
			if err := s.VerifyGPGSignature(pkgPath, sigPath); err != nil {
				return fmt.Errorf("GPG verification for %s failed: %w", pkg.Name, err)
			}
		}

		// Install
		if pkg.IsShedFormat() {
			if err := s.shed.Install(pkgPath); err != nil {
				return err
			}
		} else {
			if err := s.InstallWithPacman(pkgPath, opts); err != nil {
				return err
			}
		}
	}
	return nil
}

// Clean removes downloaded package files
func (s *ShedOSInstaller) Clean() error {
	return os.RemoveAll(s.cacheDir)
}
