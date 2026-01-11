package installer

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/pkg/core/pkgdb"
)

func TestShedOSInstaller_DownloadMultiple_Parallel(t *testing.T) {
	// 1. Setup mock server with delay
	delay := 100 * time.Millisecond
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(delay)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("fake-content"))
	}))
	defer server.Close()

	// 2. Setup Installer with Parallel=3
	cfg := config.Default()
	cfg.Network.ParallelDownloads = 3
	// Mock backend not needed for download
	installer := NewShedOSInstallerWithConfig(cfg)
	tmpDir := t.TempDir()
	installer.SetCacheDir(tmpDir)

	// 3. Define packages
	pkgs := []pkgdb.PackageInfo{
		{Name: "pkg1", Version: "1.0", DownloadURL: server.URL + "/pkg1"},
		{Name: "pkg2", Version: "1.0", DownloadURL: server.URL + "/pkg2"},
		{Name: "pkg3", Version: "1.0", DownloadURL: server.URL + "/pkg3"},
	}

	// 4. Measure time
	start := time.Now()
	results, err := installer.DownloadMultiple(pkgs, nil)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("DownloadMultiple failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// 5. Assert duration
	// If sequential: ~300ms
	// If parallel: ~100ms
	// Allow some buffer, but it should be definitely < 250ms
	if duration > 250*time.Millisecond {
		t.Errorf("Expected parallel download (approx %v), took %v. Parallelism not working?", delay, duration)
	}
}
