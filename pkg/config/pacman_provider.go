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
	"github.com/theshedman/shedman/pkg/executor"
)

// PacmanSourceProvider implements SourceProvider using the local pacman cache.
type PacmanSourceProvider struct {
	cache    *core.PackageFileCache
	engine   *core.Engine
	cacheDir string
	executor executor.Executor
}

// NewPacmanSourceProvider creates a new provider.
func NewPacmanSourceProvider(engine *core.Engine) *PacmanSourceProvider {
	return NewPacmanSourceProviderWithExecutor(engine, &executor.RealExecutor{})
}

// NewPacmanSourceProviderWithExecutor creates a provider with custom executor for testing.
func NewPacmanSourceProviderWithExecutor(engine *core.Engine, exec executor.Executor) *PacmanSourceProvider {

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
		return nil, fmt.Errorf("file %s is not owned by any known package", filePath)
	}

	// 2. Get Installed Version
	// Use cache directories to find the archive
	cacheDirs, err := p.getCacheDirs()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache dirs: %w", err)
	}

	// Get package version
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

func (p *PacmanSourceProvider) getArchitecture() (string, error) {
	out, err := p.executor.Output("uname", "-m")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (p *PacmanSourceProvider) findArchive(dirs []string, name, version string) (string, error) {
	// Detect architecture
	arch, err := p.getArchitecture()
	if err != nil {
		arch = "x86_64"
	}

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

// MaxConfigSize is the maximum size of a config file we'll extract (10MB)
const MaxConfigSize = 10 * 1024 * 1024

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
	// Add xz support if needed

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
			// Check header size first (optimization)
			if header.Size > MaxConfigSize {
				return nil, fmt.Errorf("config file %s exceeds max size limit (%d bytes)", relPath, MaxConfigSize)
			}

			// Read with limit to be safe against trailing zeroes or attacks
			var buf bytes.Buffer
			// Pre-allocate if size is reasonable
			if header.Size > 0 {
				buf.Grow(int(header.Size))
			}

			lr := io.LimitReader(tr, MaxConfigSize+1)
			n, err := io.Copy(&buf, lr)
			if err != nil {
				return nil, err
			}
			if n > MaxConfigSize {
				return nil, fmt.Errorf("config file %s exceeds max size limit", relPath)
			}

			return buf.Bytes(), nil
		}
	}

	return nil, fmt.Errorf("file %s not found in archive", relPath)
}
