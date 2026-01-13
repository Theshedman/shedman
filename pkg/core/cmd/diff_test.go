package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

func TestRunDiff(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	mock.DiffFunc = func() ([]core.PackageDiff, error) {
		return []core.PackageDiff{
			{
				Name:         "linux",
				OldVersion:   "6.1.1-1",
				NewVersion:   "6.1.2-1",
				DownloadSize: 120 * 1024 * 1024,
				SizeDelta:    512 * 1024, // +512KB
				CVEs:         []string{},
			},
			{
				Name:         "openssl",
				OldVersion:   "3.0.7-1",
				NewVersion:   "3.0.8-1",
				DownloadSize: 5 * 1024 * 1024,
				SizeDelta:    0,
				CVEs:         []string{"CVE-2023-0286"},
			},
		}, nil
	}

	var buf bytes.Buffer
	err := RunDiff(eng, &buf)
	if err != nil {
		t.Fatalf("RunDiff failed: %v", err)
	}

	out := buf.String()
	// Check headers/content
	if !strings.Contains(out, "linux") || !strings.Contains(out, "6.1.1-1 -> 6.1.2-1") {
		t.Errorf("Missing linux update info: %s", out)
	}
	if !strings.Contains(out, "openssl") || !strings.Contains(out, "CVE-2023-0286") {
		t.Errorf("Missing openssl CVE info: %s", out)
	}
	// Check size formatting approximately
	if !strings.Contains(out, "120.0 MiB") {
		t.Errorf("Missing download size: %s", out)
	}
}

func TestRunDiff_NoUpdates(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	mock.DiffFunc = func() ([]core.PackageDiff, error) {
		return []core.PackageDiff{}, nil
	}

	var buf bytes.Buffer
	err := RunDiff(eng, &buf)
	if err != nil {
		t.Fatalf("RunDiff failed: %v", err)
	}

	if !strings.Contains(buf.String(), "No updates found") {
		t.Errorf("Expected 'No updates found', got: %s", buf.String())
	}
}
