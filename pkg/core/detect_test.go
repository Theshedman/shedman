package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectDistro_Arch(t *testing.T) {
	content := `NAME="Arch Linux"
ID=arch
PRETTY_NAME="Arch Linux"
BUILD_ID=rolling
`
	info := testWithOsRelease(t, content)

	if info.ID != "arch" {
		t.Errorf("Expected ID 'arch', got '%s'", info.ID)
	}
	if info.Family != "arch" {
		t.Errorf("Expected Family 'arch', got '%s'", info.Family)
	}
}

func TestDetectDistro_Manjaro(t *testing.T) {
	content := `NAME="Manjaro Linux"
ID=manjaro
ID_LIKE=arch
PRETTY_NAME="Manjaro Linux"
`
	info := testWithOsRelease(t, content)

	if info.ID != "manjaro" {
		t.Errorf("Expected ID 'manjaro', got '%s'", info.ID)
	}
	if info.Family != "arch" {
		t.Errorf("Expected Family 'arch', got '%s'", info.Family)
	}
}

func TestDetectDistro_ShedOS(t *testing.T) {
	content := `NAME="ShedOS"
ID=shedos
ID_LIKE=arch
PRETTY_NAME="ShedOS"
VERSION="1.0"
`
	info := testWithOsRelease(t, content)

	if info.ID != "shedos" {
		t.Errorf("Expected ID 'shedos', got '%s'", info.ID)
	}
	if info.Family != "arch" {
		t.Errorf("Expected Family 'arch', got '%s'", info.Family)
	}
	if !info.IsShedOS {
		t.Error("Expected IsShedOS to be true")
	}
}

func TestDetectDistro_CachyOS(t *testing.T) {
	content := `NAME="CachyOS"
ID=cachyos
ID_LIKE=arch
PRETTY_NAME="CachyOS"
`
	info := testWithOsRelease(t, content)

	if info.ID != "cachyos" {
		t.Errorf("Expected ID 'cachyos', got '%s'", info.ID)
	}
	if info.Family != "arch" {
		t.Errorf("Expected Family 'arch', got '%s'", info.Family)
	}
}

func TestIsArchBased(t *testing.T) {
	// This test depends on the actual system
	// We just verify the function doesn't panic
	_ = IsArchBased()
}

func TestIsAURAvailable(t *testing.T) {
	// AUR availability should match Arch-based check
	if IsAURAvailable() != IsArchBased() {
		t.Error("IsAURAvailable should equal IsArchBased")
	}
}

// testWithOsRelease creates a temp os-release file for testing
func testWithOsRelease(t *testing.T, content string) DistroInfo {
	t.Helper()

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "os-release")

	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	return detectDistroFromFile(tmpFile)
}
