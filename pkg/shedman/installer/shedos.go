package installer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/theshedman/shedman/pkg/shedman/backend"
	pacmanBackend "github.com/theshedman/shedman/pkg/shedman/backend/pacman"
	"github.com/theshedman/shedman/pkg/shedman/config"
	"github.com/theshedman/shedman/pkg/shedman/pkgdb"
)

// DownloadProgressCallback reports download progress
type DownloadProgressCallback func(downloaded, total int64, speed float64)

// ShedOSInstaller handles downloading and installing ShedOS packages
type ShedOSInstaller struct {
	executor   Executor
	cacheDir   string
	httpClient *http.Client
	timeout    time.Duration
	retries    int
	shed       *ShedInstaller
	backend    backend.OfficialBackend // Backend for local package installation
}

// NewShedOSInstaller creates a new ShedOSInstaller with default config
func NewShedOSInstaller() *ShedOSInstaller {
	return NewShedOSInstallerWithConfig(config.Default())
}

// NewShedOSInstallerWithConfig creates a new ShedOSInstaller with config
// This auto-detects the backend; use NewShedOSInstallerWithBackend for explicit injection
func NewShedOSInstallerWithConfig(cfg *config.Config) *ShedOSInstaller {
	// Auto-detect backend
	var b backend.OfficialBackend
	if pacmanBackend.IsPacmanAvailable() {
		b, _ = pacmanBackend.New()
	}
	return NewShedOSInstallerWithBackend(cfg, b)
}

// NewShedOSInstallerWithBackend creates a new ShedOSInstaller with explicit backend injection
// This is the preferred constructor for production use and testing
func NewShedOSInstallerWithBackend(cfg *config.Config, b backend.OfficialBackend) *ShedOSInstaller {
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

	return &ShedOSInstaller{
		executor:   DefaultExecutor,
		cacheDir:   cacheDir,
		httpClient: &http.Client{Timeout: timeout},
		timeout:    timeout,
		retries:    retries,
		shed:       NewShedInstaller(),
		backend:    b,
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
func (s *ShedOSInstaller) Download(pkg pkgdb.PackageInfo) (string, error) {
	return s.DownloadWithProgress(pkg, nil)
}

// DownloadWithProgress downloads a package with progress reporting
func (s *ShedOSInstaller) DownloadWithProgress(pkg pkgdb.PackageInfo, callback DownloadProgressCallback) (string, error) {
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
				speed := float64(downloaded) / elapsed / 1024 // KB/s
				callback(downloaded, totalSize, speed)
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
func (s *ShedOSInstaller) DownloadSignature(pkg pkgdb.PackageInfo) (string, error) {
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
	return s.executor(cmd)
}

// Install installs a package from ShedOS repository
func (s *ShedOSInstaller) Install(pkg pkgdb.PackageInfo, opts Options) error {
	return s.InstallWithProgress(pkg, opts, nil)
}

// InstallWithProgress installs a package with progress callbacks
func (s *ShedOSInstaller) InstallWithProgress(pkg pkgdb.PackageInfo, opts Options, callback DownloadProgressCallback) error {
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
	// Use backend if available for local package installation
	if s.backend != nil {
		return s.backend.InstallLocal(pkgPath, ToBackendOptions(opts))
	}

	// Fallback to direct pacman command
	cmd := []string{"sudo", "pacman", "-U"}

	if opts.Needed {
		cmd = append(cmd, "--needed")
	}
	if opts.AsDeps {
		cmd = append(cmd, "--asdeps")
	}
	if opts.AsExplicit {
		cmd = append(cmd, "--asexplicit")
	}
	if opts.NoConfirm {
		cmd = append(cmd, "--noconfirm")
	}

	cmd = append(cmd, pkgPath)
	return s.executor(cmd)
}

// InstallMultiple installs multiple packages from ShedOS
func (s *ShedOSInstaller) InstallMultiple(pkgs []pkgdb.PackageInfo, opts Options) error {
	for _, pkg := range pkgs {
		if err := s.Install(pkg, opts); err != nil {
			return fmt.Errorf("failed to install %s: %w", pkg.Name, err)
		}
	}
	return nil
}

// Clean removes downloaded package files
func (s *ShedOSInstaller) Clean() error {
	return os.RemoveAll(s.cacheDir)
}
