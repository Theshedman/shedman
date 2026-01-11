package installer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTransaction_TrackCreate_Rollback(t *testing.T) {
	tx, err := NewTransaction(nil)
	if err != nil {
		t.Fatalf("NewTransaction failed: %v", err)
	}

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "newfile.txt")

	// 1. Track Create
	tx.TrackCreate(file)

	// 2. Actually create it (simulate install)
	if err := os.WriteFile(file, []byte("content"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// 3. Rollback
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// 4. Verify file is gone
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Errorf("File %s should have been deleted by rollback", file)
	}
}

func TestTransaction_TrackOverwrite_Rollback(t *testing.T) {
	tx, err := NewTransaction(nil)
	if err != nil {
		t.Fatalf("NewTransaction failed: %v", err)
	}

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "existing.txt")

	// 1. Create original file
	originalContent := []byte("original")
	if err := os.WriteFile(file, originalContent, 0644); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// 2. Track Overwrite (should backup)
	if err := tx.TrackOverwrite(file); err != nil {
		t.Fatalf("TrackOverwrite failed: %v", err)
	}

	// 3. Overwrite it
	newContent := []byte("new")
	if err := os.WriteFile(file, newContent, 0644); err != nil {
		t.Fatalf("Overwrite failed: %v", err)
	}

	// 4. Rollback
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// 5. Verify restored
	content, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(content) != string(originalContent) {
		t.Errorf("Content mismatch. Got %s, want %s", content, originalContent)
	}
}

func TestTransaction_Commit(t *testing.T) {
	tx, err := NewTransaction(nil)
	if err != nil {
		t.Fatalf("NewTransaction failed: %v", err)
	}

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "committed.txt")

	tx.TrackCreate(file)
	os.WriteFile(file, []byte("data"), 0644)

	// Commit
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// File should still exist
	if _, err := os.Stat(file); os.IsNotExist(err) {
		t.Errorf("File should exist after commit")
	}

	// Backup dir should be gone
	if _, err := os.Stat(tx.backupDir); !os.IsNotExist(err) {
		t.Errorf("Backup dir should be deleted")
	}
}
