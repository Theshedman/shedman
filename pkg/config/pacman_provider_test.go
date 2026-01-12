package config

import (
	"archive/tar"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/theshedman/shedman/pkg/core"
)

// MockExecutor implements util.Executor for testing
type MockExecutor struct {
	Responses map[string]struct {
		Output []byte
		Err    error
	}
}

func (m *MockExecutor) Output(name string, args ...string) ([]byte, error) {
	key := name + " " + strings.Join(args, " ")
	// Simple key matching. In real tests we might need regex or exact args matching.
	if resp, ok := m.Responses[key]; ok {
		return resp.Output, resp.Err
	}
	return nil, fmt.Errorf("unexpected command: %s", key)
}

func TestGetOriginalContent_Success(t *testing.T) {
	// Setup temporary cache dir
	tmpDir, err := os.MkdirTemp("", "shedman-test-cache")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create dummy archive
	pkgName := "testpkg"
	pkgVer := "1.0-1"
	arch := "x86_64"
	archiveName := fmt.Sprintf("%s-%s-%s.pkg.tar.zst", pkgName, pkgVer, arch)
	archivePath := filepath.Join(tmpDir, archiveName)

	if err := createDummyArchive(archivePath, "etc/test.conf", "original content"); err != nil {
		t.Fatalf("Failed to create dummy archive: %v", err)
	}

	// Mock Executor
	mockExec := &MockExecutor{
		Responses: map[string]struct {
			Output []byte
			Err    error
		}{
			"pacman-conf CacheDir": {
				Output: []byte(tmpDir + "\n"),
				Err:    nil,
			},
			"pacman -Q testpkg": {
				Output: []byte("testpkg " + pkgVer),
				Err:    nil,
			},
		},
	}

	// Mock Cache?
	// PacmanSourceProvider uses core.GetDefaultCache() internally in `NewPacmanSourceProviderWithExecutor`?
	// Note: The implementation:
	// return &PacmanSourceProvider{
	// 	cache:    core.GetDefaultCache(),
	// ...
	// }
	// We need to inject a mock cache or seed the default cache.
	// core.PackageFileCache is struct, not interface. We can seed it.
	cache := core.GetDefaultCache()
	cache.SetPackageFiles(pkgName, []string{"/etc/test.conf"})

	// Create Provider
	p := NewPacmanSourceProviderWithExecutor(nil, mockExec)

	// Test
	content, err := p.GetOriginalContent("/etc/test.conf")
	if err != nil {
		t.Fatalf("GetOriginalContent failed: %v", err)
	}

	if string(content) != "original content" {
		t.Errorf("Expected 'original content', got '%s'", string(content))
	}
}

func createDummyArchive(path, filename, content string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Zstd writer
	zw, _ := zstd.NewWriter(f)
	defer zw.Close()

	// Tar writer
	tw := tar.NewWriter(zw)
	defer tw.Close()

	body := []byte(content)
	hdr := &tar.Header{
		Name: filename,
		Mode: 0644,
		Size: int64(len(body)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	if _, err := tw.Write(body); err != nil {
		return err
	}
	return nil
}
