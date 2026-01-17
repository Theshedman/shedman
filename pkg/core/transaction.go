package core

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// OperationType tracks the type of file operation
type OperationType int

const (
	OpCreated OperationType = iota
	OpOverwritten
	OpDirectoryCreated // Track directories created so we can remove them if empty on rollback
)

// JournalEntry is a record in the transaction log
type JournalEntry struct {
	Type       OperationType
	Path       string
	BackupPath string
}

// Transaction manages file system changes and supports rollback
type Transaction struct {
	entries   []JournalEntry
	backupDir string
	active    bool
	executor  func(string, []string) error
}

// NewTransaction creates a new transaction
func NewTransaction(executor func(string, []string) error) (*Transaction, error) {
	// Create a temp directory for backups
	backupDir, err := os.MkdirTemp("", "shedman-tx-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction backup dir: %w", err)
	}

	return &Transaction{
		entries:   make([]JournalEntry, 0),
		backupDir: backupDir,
		active:    true,
		executor:  executor,
	}, nil
}

// TrackCreate records that a file is about to be created
// It does NOT create the file, just logs the intent for rollback
func (t *Transaction) TrackCreate(path string) {
	if !t.active {
		return
	}
	t.entries = append(t.entries, JournalEntry{
		Type: OpCreated,
		Path: path,
	})
}

// TrackDirectoryCreate records that a directory was created
func (t *Transaction) TrackDirectoryCreate(path string) {
	if !t.active {
		return
	}
	t.entries = append(t.entries, JournalEntry{
		Type: OpDirectoryCreated,
		Path: path,
	})
}

// TrackOverwrite backs up an existing file and records the overwrite
func (t *Transaction) TrackOverwrite(path string) error {
	if !t.active {
		return nil
	}

	// Verify source exists
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		t.TrackCreate(path)
		return nil
	}

	// Create backup
	backupName := fmt.Sprintf("backup-%d", len(t.entries))
	backupPath := filepath.Join(t.backupDir, backupName)

	// Use executor (sudo) to copy to backup if needed
	if t.executor != nil {
		cmd := []string{"sudo", "cp", "-p", path, backupPath}
		if err := t.executor("", cmd); err != nil {
			return fmt.Errorf("failed to backup file %s using sudo: %w", path, err)
		}
	} else {
		if err := copyFile(path, backupPath); err != nil {
			return fmt.Errorf("failed to backup file for transaction: %w", err)
		}
	}

	t.entries = append(t.entries, JournalEntry{
		Type:       OpOverwritten,
		Path:       path,
		BackupPath: backupPath,
	})

	return nil
}

// Rollback reverts all changes in reverse order
func (t *Transaction) Rollback() error {
	if !t.active {
		return nil
	}
	t.active = false // prevent further tracking

	var lastErr error

	// Iterate backwards
	for i := len(t.entries) - 1; i >= 0; i-- {
		entry := t.entries[i]
		switch entry.Type {
		case OpCreated:
			// Remove the created file
			if t.executor != nil {
				_ = t.executor("", []string{"sudo", "rm", "-f", entry.Path})

			} else {
				os.Remove(entry.Path)
			}
		case OpDirectoryCreated:
			// Remove directory
			if t.executor != nil {
				_ = t.executor("", []string{"sudo", "rmdir", entry.Path})

			} else {
				os.Remove(entry.Path)
			}
		case OpOverwritten:
			// Restore from backup
			if t.executor != nil {
				_ = t.executor("", []string{"sudo", "cp", "-p", entry.BackupPath, entry.Path})

			} else {
				if err := copyFile(entry.BackupPath, entry.Path); err != nil {
					lastErr = fmt.Errorf("failed to restore %s: %w", entry.Path, err)
				}
			}
		}
	}

	// Cleanup backup dir
	os.RemoveAll(t.backupDir)

	return lastErr
}

// Commit finalizes the transaction (deletes backups)
func (t *Transaction) Commit() error {
	if !t.active {
		return nil
	}
	t.active = false
	return os.RemoveAll(t.backupDir)
}

// Helper to copy file
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	// Preserve permissions
	info, err := in.Stat()
	if err == nil {
		_ = os.Chmod(dst, info.Mode())

	}

	return nil
}
