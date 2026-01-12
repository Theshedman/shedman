package config

import (
	"archive/tar"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
	"github.com/theshedman/shedman/internal/util"
	"github.com/theshedman/shedman/pkg/core"
)

// PacmanSourceProvider implements SourceProvider using the local pacman cache.
type PacmanSourceProvider struct {
	cache    *core.PackageFileCache
	engine   *core.Engine
	cacheDir string
	executor util.Executor
}

// NewPacmanSourceProvider creates a new provider.
func NewPacmanSourceProvider(engine *core.Engine) *PacmanSourceProvider {
	return NewPacmanSourceProviderWithExecutor(engine, &util.RealExecutor{})
}

// NewPacmanSourceProviderWithExecutor creates a provider with custom executor for testing.
func NewPacmanSourceProviderWithExecutor(engine *core.Engine, exec util.Executor) *PacmanSourceProvider {
	return &PacmanSourceProvider{
		cache:    core.GetDefaultCache(),
		engine:   engine,
		executor: exec,
	}
}

// GetOriginalContent retrieves content from the package archive.
func (p *PacmanSourceProvider) GetOriginalContent(filePath string) ([]byte, error) {
	// 1. Identify Owner
	owner, found := p.cache.GetFileOwner(filePath)
	if !found {
		// Try refreshing cache for this file or just fail if not tracked?
		// User might request diff on a file not yet loaded in cache.
		// Try to resolve owner dynamically if possible, or assume cache is populated.
		// For now, assume cache is populated or fail.
		// Since we want robust, let's try to query pacman -Qo if cache fails?
		// But core/cache is the source of truth for us.
		return nil, fmt.Errorf("file %s is not owned by any known package", filePath)
	}

	// 2. Get Installed Version
	// We need the exact version to find the archive.
	// 2. Get Installed Version
	// We use exec for robustness

	// We need a method to get specific package info from Core/Backend without overhead.
	// p.engine.Info() might work if exposed.
	// Or we can use `pacman -Q <pkg>` but that's what backend does.
	// Since engine.Info might not be exposed efficiently for this, let's assume we can use a helper or backend directly.
	// backend := p.engine.GetOfficialBackend()
	// But getting backend requires type assertion.
	// Lets stick to `pacman -Q` explicitly for robustness here, or assume we can find the version.

	// Actually, let's use `pacman -Q <pkg>` via exec for simplicity and reliability if backend API is complex.
	// But we should use the backend if possible.
	// Let's rely on `pacman -Q` output parsing for now, or better:
	// Find the archive by globbing the cache dir?
	// Archives are named: name-version-release-arch.pkg.tar.zst

	// Let's first get the CacheDir.
	cacheDirs, err := p.getCacheDirs()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache dirs: %w", err)
	}

	// We need the full version string (ver-rel).
	version, err := p.getPackageVersion(owner)
	if err != nil {
		return nil, fmt.Errorf("failed to get package version for %s: %w", owner, err)
	}

	// 3. Locate Archive
	archivePath, err := p.findArchive(cacheDirs, owner, version)
	if err != nil {
		return nil, fmt.Errorf("archive not found for %s %s: %w", owner, version, err)
	}

	// 4. Extract Content
	// File path in archive usually doesn't have leading /.
	relPath := strings.TrimPrefix(filePath, "/")

	return p.extractFile(archivePath, relPath)
}

// GetOwner returns the package name owning the file.
func (p *PacmanSourceProvider) GetOwner(filePath string) (string, error) {
	owner, found := p.cache.GetFileOwner(filePath)
	if !found {
		return "", fmt.Errorf("file %s is not owned by any known package", filePath)
	}
	return owner, nil
}

func (p *PacmanSourceProvider) getCacheDirs() ([]string, error) {
	if p.cacheDir != "" {
		return []string{p.cacheDir}, nil
	}

	out, err := p.executor.Output("pacman-conf", "CacheDir")
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(out))
	var dirs []string
	for scanner.Scan() {
		dir := strings.TrimSpace(scanner.Text())
		if dir != "" {
			dirs = append(dirs, dir)
		}
	}
	// Fallback
	if len(dirs) == 0 {
		dirs = []string{"/var/cache/pacman/pkg/"}
	}
	return dirs, nil
}

func (p *PacmanSourceProvider) getPackageVersion(pkgName string) (string, error) {
	// Use pacman -Q to get version
	out, err := p.executor.Output("pacman", "-Q", pkgName)
	if err != nil {
		return "", err
	}
	// Output: package version
	parts := strings.Fields(string(out))
	if len(parts) >= 2 {
		return parts[1], nil
	}
	return "", fmt.Errorf("unexpected output from pacman -Q")
}

func (p *PacmanSourceProvider) findArchive(dirs []string, name, version string) (string, error) {
	// Standard Arch naming: name-version-arch.pkg.tar.zst
	// We need architecture too? `uname -m`
	arch := "x86_64" // Assume x86_64 for now, or detect
	// Actually, just globbing name-version-* might be safer.

	extensions := []string{".pkg.tar.zst", ".pkg.tar.xz"}

	for _, dir := range dirs {
		for _, ext := range extensions {
			// Try specific format first
			fname := fmt.Sprintf("%s-%s-%s%s", name, version, arch, ext)
			path := filepath.Join(dir, fname)
			if exists, _ := util.FileExists(path); exists {
				return path, nil
			}

			// Fallback: try finding 'any' arch
			fnameAny := fmt.Sprintf("%s-%s-any%s", name, version, ext)
			pathAny := filepath.Join(dir, fnameAny)
			if exists, _ := util.FileExists(pathAny); exists {
				return pathAny, nil
			}
		}
	}
	return "", os.ErrNotExist
}

func (p *PacmanSourceProvider) extractFile(archivePath, relPath string) ([]byte, error) {
	// Use external tar if possible for robustness, or Go's archive/tar + zstd.
	// Go's zstd support is good.

	f, err := os.Open(archivePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var r io.Reader = f

	// Decompress
	if strings.HasSuffix(archivePath, ".zst") {
		decoder, err := zstd.NewReader(f)
		if err != nil {
			return nil, err
		}
		defer decoder.Close()
		r = decoder
	}
	// Add xz support if needed (not in stdlib, assume zstd for now as it's standard)

	tr := tar.NewReader(r)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if header.Name == relPath {
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, tr); err != nil {
				return nil, err
			}
			return buf.Bytes(), nil
		}
	}

	return nil, fmt.Errorf("file %s not found in archive", relPath)
}
