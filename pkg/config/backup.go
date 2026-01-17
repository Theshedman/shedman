package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// FileBackupManager implements BackupManager using simple file copies
type FileBackupManager struct{}

// NewFileBackupManager creates a new backup manager
func NewFileBackupManager() *FileBackupManager {
	return &FileBackupManager{}
}

// Backup creates a timestamped backup of the file at path
func (m *FileBackupManager) Backup(path string) (string, error) {
	// Check if source exists
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("cannot backup directory: %s", path)
	}

	// Generate backup path
	timestamp := time.Now().Format("20060102150405.000") // YYYYMMDDHHMMSS.mss
	backupPath := fmt.Sprintf("%s.%s.bak", path, timestamp)

	// Copy content
	source, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open source file: %w", err)
	}
	defer func() { _ = source.Close() }()

	dest, err := os.OpenFile(backupPath, os.O_RDWR|os.O_CREATE|os.O_EXCL, info.Mode())
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}
	defer func() { _ = dest.Close() }()

	if _, err := io.Copy(dest, source); err != nil {
		return "", fmt.Errorf("failed to copy file content: %w", err)
	}

	return backupPath, nil
}

// Rotate keeps only the last N backups for a given file
func (m *FileBackupManager) Rotate(path string, keep int) error {
	if keep < 1 {
		return nil
	}

	// Pattern: path.<timestamp>.bak
	pattern := path + ".*.bak"
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to glob backups: %w", err)
	}

	if len(matches) <= keep {
		return nil
	}

	// Sort matches so oldest are first
	sort.Strings(matches)

	// Determine which to delete
	// If we have N matches and want to keep K, we delete N-K oldest.
	// Since ISO timestamps sort lexicographically, oldest are at the beginning.
	numToDelete := len(matches) - keep
	toDelete := matches[:numToDelete]

	for _, p := range toDelete {
		if err := os.Remove(p); err != nil {
			return fmt.Errorf("failed to remove old backup %s: %w", p, err)
		}
	}

	return nil
}
