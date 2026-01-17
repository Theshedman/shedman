package config

import (
	"archive/tar"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/theshedman/shedman/internal/util"
	"github.com/theshedman/shedman/pkg/core"
)

// MockExecutor implements util.Executor for testing
type MockExecutor struct {
	Responses map[string]struct {
		Output []byte
		Err    error
	}
	// OutputFunc allows for custom mocking of the Output method
	OutputFunc func(name string, args ...string) ([]byte, error)
}

func (m *MockExecutor) Output(name string, args ...string) ([]byte, error) {
	if m.OutputFunc != nil {
		return m.OutputFunc(name, args...)
	}
	key := name + " " + strings.Join(args, " ")
	// Simple key matching.
	if resp, ok := m.Responses[key]; ok {
		return resp.Output, resp.Err
	}
	return nil, fmt.Errorf("unexpected command: %s", key)
}

func (m *MockExecutor) Command(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

func (m *MockExecutor) CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}

func TestGetOriginalContent_Success(t *testing.T) {
	// Setup temporary cache dir
	tmpDir, err := os.MkdirTemp("", "shedman-test-cache")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

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
			"uname -m": {
				Output: []byte(arch + "\n"),
				Err:    nil,
			},
		},
	}

	// Seed cache
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
	defer func() { _ = f.Close() }()

	// Zstd writer
	zw, _ := zstd.NewWriter(f)
	defer func() { _ = zw.Close() }()

	// Tar writer
	tw := tar.NewWriter(zw)
	defer func() { _ = tw.Close() }()

	body := []byte(content)
	hdr := &tar.Header{
		Name: filename,
		Mode: int64(util.FilePermissions),
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
