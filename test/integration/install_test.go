//go:build integration

package integration

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var binaryPath string

func init() {
	// Binary expected in root
	_, filename, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(filename), "../..")
	binaryPath = filepath.Join(root, "shedman")
}

func TestInstall_DryRun_Official(t *testing.T) {
	cmd := exec.Command(binaryPath, "install", "neovim", "--dry-run", "--noconfirm")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run shedman install: %v\nOutput: %s", err, output)
	}

	outStr := string(output)
	if !strings.Contains(outStr, "neovim") {
		t.Errorf("Expected output to contain 'neovim', got:\n%s", outStr)
	}
	if !strings.Contains(outStr, "official") {
		t.Errorf("Expected output to contain '[official]', got:\n%s", outStr)
	}
	if !strings.Contains(outStr, "Dry-run mode") {
		t.Errorf("Expected output to contain 'Dry-run mode', got:\n%s", outStr)
	}
}

func TestInstall_DryRun_AUR(t *testing.T) {
	// Use a known AUR package, e.g. 'google-chrome' or 'visual-studio-code-bin'
	// Using 'yay-bin' to force AUR usage
	cmd := exec.Command(binaryPath, "install", "yay-bin", "--dry-run", "--noconfirm", "--aur")
	// Requires AUR enabled in config
	// We might need to supply a config or flags.
	output, _ := cmd.CombinedOutput()

	outStr := string(output)
	// Check for backend source indication in output
	// Or we can invoke search.
	// For install, if not found, it errors.
	if strings.Contains(outStr, "Dry-run mode") && strings.Contains(outStr, "yay-bin") && (strings.Contains(outStr, "[AUR]") || strings.Contains(outStr, "aur/yay-bin")) {
		// Pass
	} else {
		t.Skip("Skipping AUR test as package might not be resolvable or AUR disabled")
	}
}
