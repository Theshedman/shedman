package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/theshedman/shedman/internal/config"
)

type loggingBackend struct{}

func (l *loggingBackend) Name() string                                { return "logging" }
func (l *loggingBackend) Sync() error                                 { return nil }
func (l *loggingBackend) IsAvailable() bool                           { return true }
func (l *loggingBackend) Install(pkgs []string, opts InstallOptions) error {
	return nil
}
func (l *loggingBackend) Remove(pkgs []string, opts RemoveOptions) error { return nil }
func (l *loggingBackend) IsInstalled(pkgName string) bool                { return false }
func (l *loggingBackend) Search(query string) ([]PackageInfo, error)     { return nil, nil }
func (l *loggingBackend) Info(pkgName string) (*PackageInfo, error)      { return nil, nil }
func (l *loggingBackend) GetInstalledPackages() ([]PackageInfo, error)   { return nil, nil }
func (l *loggingBackend) Upgrade(pkgs []string, opts UpgradeOptions) error {
	return nil
}
func (l *loggingBackend) InstallLocal(path string, opts InstallOptions) error {
	return nil
}
func (l *loggingBackend) GetPackageFiles(pkgName string) ([]string, error) { return nil, nil }
func (l *loggingBackend) GetFileOwner(path string) (string, error)         { return "", nil }
func (l *loggingBackend) SearchFiles(query string) ([]string, error)       { return nil, nil }
func (l *loggingBackend) ListExplicitPackages() ([]string, error)          { return nil, nil }
func (l *loggingBackend) Audit() ([]string, error)                         { return nil, nil }
func (l *loggingBackend) Diff() ([]PackageDiff, error)                     { return nil, nil }

func TestEngine_LogsInstall(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "shedman.log")

	engine := NewEngineWithBackend(&loggingBackend{})
	engine.SetConfig(&config.Config{
		Logging: config.LoggingConfig{
			Enabled: true,
			File:    logPath,
		},
	})

	if err := engine.Install([]string{"vim"}, InstallOptions{}); err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log failed: %v", err)
	}

	output := string(data)
	if !strings.Contains(output, "installed vim") {
		t.Errorf("expected log entry for install, got: %s", output)
	}

	// Verify timestamp format exists
	if idx := strings.Index(output, "]"); idx > 1 {
		if _, err := time.Parse("2006-01-02T15:04:05-0700", output[1:idx]); err != nil {
			t.Errorf("expected timestamp format, got: %v", err)
		}
	} else {
		t.Errorf("unexpected log format: %s", output)
	}
}
