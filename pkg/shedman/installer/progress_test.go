package installer_test

import (
	"testing"

	"github.com/theshedman/shedman/pkg/shedman/installer"
)

func TestPacmanProgress_ParseDownloading(t *testing.T) {
	var events []installer.ProgressEvent
	callback := func(e installer.ProgressEvent) {
		events = append(events, e)
	}

	pp := installer.NewPacmanProgress(callback)
	pp.ParseLine("downloading neovim...")

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}
	if events[0].Type != installer.ProgressDownloading {
		t.Errorf("Expected ProgressDownloading, got %d", events[0].Type)
	}
	if events[0].Package != "neovim" {
		t.Errorf("Expected package 'neovim', got %s", events[0].Package)
	}
}

func TestPacmanProgress_ParseInstalling(t *testing.T) {
	var events []installer.ProgressEvent
	callback := func(e installer.ProgressEvent) {
		events = append(events, e)
	}

	pp := installer.NewPacmanProgress(callback)
	pp.ParseLine("installing git...")

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}
	if events[0].Type != installer.ProgressInstalling {
		t.Errorf("Expected ProgressInstalling, got %d", events[0].Type)
	}
}

func TestFilesystemScanner_New(t *testing.T) {
	scanner := installer.NewFilesystemScanner("/")
	if scanner == nil {
		t.Error("Expected non-nil scanner")
	}
}

func TestFilesystemScanner_SetMaxDepth(t *testing.T) {
	scanner := installer.NewFilesystemScanner("/")
	scanner.SetMaxDepth(5)
	// No panic = success
}

func TestFileScanResult_Empty(t *testing.T) {
	result := &installer.FileScanResult{
		Files: make(map[string]installer.FileInfo),
	}
	if len(result.Files) != 0 {
		t.Error("Expected empty files map")
	}
}

func TestCheckFileConflicts_NoConflicts(t *testing.T) {
	packageFiles := []string{"/usr/bin/newcmd"}
	existingFiles := make(map[string]installer.FileInfo)

	conflicts := installer.CheckFileConflicts(packageFiles, existingFiles)
	if len(conflicts) != 0 {
		t.Errorf("Expected no conflicts, got %d", len(conflicts))
	}
}

func TestCheckFileConflicts_WithConflict(t *testing.T) {
	packageFiles := []string{"/usr/bin/vim", "/usr/bin/newcmd"}
	existingFiles := map[string]installer.FileInfo{
		"/usr/bin/vim": {Path: "/usr/bin/vim"},
	}

	conflicts := installer.CheckFileConflicts(packageFiles, existingFiles)
	if len(conflicts) != 1 {
		t.Errorf("Expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0] != "/usr/bin/vim" {
		t.Errorf("Expected conflict '/usr/bin/vim', got %s", conflicts[0])
	}
}
