package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateManager_LoadSave(t *testing.T) {
	// Setup temporary state directory
	tmpDir, err := os.MkdirTemp("", "shedman-state-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	stateFile := filepath.Join(tmpDir, "config_state.json")

	// 1. Initialize new manager (should start empty)
	manager := NewJSONStateManager(stateFile)
	err = manager.Load()
	require.NoError(t, err, "Load on missing file should succeed (init empty)")

	// 2. Set some state
	testState := FileState{
		Path:         "config/hypr/hyprland.conf",
		Hash:         "sha256:123456",
		Version:      "1.0.0",
		LastModified: time.Now().UTC(),
	}
	manager.Set("hypr", "config/hypr/hyprland.conf", testState)

	// Verify in-memory get
	got, found := manager.Get("hypr", "config/hypr/hyprland.conf")
	assert.True(t, found)
	assert.Equal(t, testState.Hash, got.Hash)

	// 3. Save to disk
	err = manager.Save()
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(stateFile)
	assert.NoError(t, err)

	// 4. Load from disk with new manager instance
	manager2 := NewJSONStateManager(stateFile)
	err = manager2.Load()
	require.NoError(t, err)

	got2, found2 := manager2.Get("hypr", "config/hypr/hyprland.conf")
	assert.True(t, found2)
	assert.Equal(t, testState.Hash, got2.Hash)
	assert.Equal(t, testState.Version, got2.Version)
}

func TestStateManager_AtomicWrite(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "shedman-state-atomic")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	stateFile := filepath.Join(tmpDir, "state.json")
	manager := NewJSONStateManager(stateFile)

	err = manager.Save()
	require.NoError(t, err)

	// Verify temp file is gone
	files, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, 1, len(files), "Only the final state file should remain")
	assert.Equal(t, "state.json", files[0].Name())
}
