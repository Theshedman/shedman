package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theshedman/shedman/internal/util"
)

func TestBackupManager_Backup(t *testing.T) {
	// Setup temp dir
	tmpDir, err := os.MkdirTemp("", "shedman-backup-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a dummy config file
	configFile := filepath.Join(tmpDir, "test.conf")
	err = os.WriteFile(configFile, []byte("original content"), util.FilePermissions)
	require.NoError(t, err)

	// Initialize manager
	manager := NewFileBackupManager()

	// 1. Create a backup
	backupPath, err := manager.Backup(configFile)
	require.NoError(t, err)

	// Verify backup exists
	_, err = os.Stat(backupPath)
	assert.NoError(t, err)
	assert.NotEqual(t, configFile, backupPath)

	// Verify content matches
	content, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	assert.Equal(t, "original content", string(content))
}

func TestBackupManager_Rotate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "shedman-rotate-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	configFile := filepath.Join(tmpDir, "rotate.conf")
	err = os.WriteFile(configFile, []byte("data"), util.FilePermissions)
	require.NoError(t, err)

	manager := NewFileBackupManager()

	// Create 5 backups manually with different timestamps to simulate history
	// Create 5 backups manually
	// We rely on the implementation ensuring unique timestamps

	var backups []string
	for i := 0; i < 6; i++ {
		path, err := manager.Backup(configFile)
		require.NoError(t, err)
		backups = append(backups, path)
		// Ensure unique usage of time
		time.Sleep(10 * time.Millisecond)
	}

	// Verify we have 6 backups
	pattern := configFile + ".*.bak"
	matches, _ := filepath.Glob(pattern)
	require.Equal(t, 6, len(matches))

	// Rotate to keep 3
	err = manager.Rotate(configFile, 3)
	require.NoError(t, err)

	// Verify only 3 remain
	matches, _ = filepath.Glob(pattern)
	assert.Equal(t, 3, len(matches))

	// Verify the OLDEST ones were deleted (first 3 in our list)
	for i := 0; i < 3; i++ {
		_, err := os.Stat(backups[i])
		assert.True(t, os.IsNotExist(err), "Old backup %s should be deleted", backups[i])
	}
	// Verify NEWEST ones remain (last 3)
	for i := 3; i < 6; i++ {
		_, err := os.Stat(backups[i])
		assert.NoError(t, err, "New backup %s should exist", backups[i])
	}
}

func TestBackupManager_BackupMissingFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "shedman-backup-missing")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	manager := NewFileBackupManager()
	_, err = manager.Backup(filepath.Join(tmpDir, "missing.conf"))
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}
