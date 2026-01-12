package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockConflictResolver
type MockConflictResolver struct {
	mock.Mock
}

func (m *MockConflictResolver) Resolve(file string, diff string) (Action, error) {
	args := m.Called(file, diff)
	return args.Get(0).(Action), args.Error(1)
}

// Helper to setup engine with temporary paths
func setupEngineTest(t *testing.T) (*ConfigEngine, string, string) {
	tmpDir, err := os.MkdirTemp("", "shedman-engine-test")
	require.NoError(t, err)

	stateFile := filepath.Join(tmpDir, "state.json")
	stateMgr := NewJSONStateManager(stateFile)
	backupMgr := NewFileBackupManager()
	differ := NewDiffer()
	resolver := new(MockConflictResolver)

	engine := NewConfigEngine(stateMgr, backupMgr, differ, resolver)

	return engine, tmpDir, stateFile
}

func TestEngine_DecisionMatrix(t *testing.T) {
	// Scenario 1: Fresh Install (Target Missing)
	t.Run("FreshInstall", func(t *testing.T) {
		eng, tmpDir, _ := setupEngineTest(t)
		defer os.RemoveAll(tmpDir)

		src := filepath.Join(tmpDir, "pkg.conf")
		os.WriteFile(src, []byte("pkg content"), 0644)
		target := filepath.Join(tmpDir, "user.conf")

		// Apply
		err := eng.Apply("testpkg", src, target)
		require.NoError(t, err)

		// Verify copied
		content, _ := os.ReadFile(target)
		assert.Equal(t, "pkg content", string(content))

		// Verify state updated
		state, found := eng.stateMgr.Get("testpkg", target)
		assert.True(t, found)
		hash, _ := eng.differ.CalculateHash(src)
		assert.Equal(t, hash, state.Hash)
	})

	// Scenario 2: No Changes (All Match)
	t.Run("NoChanges", func(t *testing.T) {
		eng, tmpDir, _ := setupEngineTest(t)
		defer os.RemoveAll(tmpDir)

		content := []byte("content")
		src := filepath.Join(tmpDir, "pkg.conf")
		target := filepath.Join(tmpDir, "user.conf")
		os.WriteFile(src, content, 0644)
		os.WriteFile(target, content, 0644)

		// Set State match
		hash := eng.differ.CalculateStringHash(string(content))
		eng.stateMgr.Set("testpkg", target, FileState{
			Path: target, Hash: hash, LastModified: time.Now(),
		})

		err := eng.Apply("testpkg", src, target)
		require.NoError(t, err)

		// Ensure file untouched (modtime check ideally, but content check enough)
		newContent, _ := os.ReadFile(target)
		assert.Equal(t, string(content), string(newContent))
	})

	// Scenario 3: User Only Change -> Keep User (Default)
	t.Run("UserChange_Keep", func(t *testing.T) {
		eng, tmpDir, _ := setupEngineTest(t)
		defer os.RemoveAll(tmpDir)

		baseContent := "base"
		userContent := "user modified"
		pkgContent := "base" // Package matches base

		src := filepath.Join(tmpDir, "pkg.conf")
		target := filepath.Join(tmpDir, "user.conf")
		os.WriteFile(src, []byte(pkgContent), 0644)
		os.WriteFile(target, []byte(userContent), 0644)

		// State matches base
		baseHash := eng.differ.CalculateStringHash(baseContent)
		eng.stateMgr.Set("testpkg", target, FileState{Path: target, Hash: baseHash})

		err := eng.Apply("testpkg", src, target)
		require.NoError(t, err)

		// Verify USER content kept
		finalContent, _ := os.ReadFile(target)
		assert.Equal(t, userContent, string(finalContent))
	})

	// Scenario 4: Package Update -> Update
	t.Run("PackageUpdate", func(t *testing.T) {
		eng, tmpDir, _ := setupEngineTest(t)
		defer os.RemoveAll(tmpDir)

		baseContent := "base"
		userContent := "base" // User matches base
		pkgContent := "new version"

		src := filepath.Join(tmpDir, "pkg.conf")
		target := filepath.Join(tmpDir, "user.conf")
		os.WriteFile(src, []byte(pkgContent), 0644)
		os.WriteFile(target, []byte(userContent), 0644)

		baseHash := eng.differ.CalculateStringHash(baseContent)
		eng.stateMgr.Set("testpkg", target, FileState{Path: target, Hash: baseHash})

		err := eng.Apply("testpkg", src, target)
		require.NoError(t, err)

		// Verify Updated
		finalContent, _ := os.ReadFile(target)
		assert.Equal(t, pkgContent, string(finalContent))

		// Verify State Updated
		newState, _ := eng.stateMgr.Get("testpkg", target)
		newHash := eng.differ.CalculateStringHash(pkgContent)
		assert.Equal(t, newHash, newState.Hash)
	})

	// Scenario 5: Conflict -> Interactive (Mocked: Keep User)
	t.Run("Conflict_KeepUser", func(t *testing.T) {
		eng, tmpDir, _ := setupEngineTest(t)
		defer os.RemoveAll(tmpDir)

		baseContent := "base"
		userContent := "user change"
		pkgContent := "pkg change"

		src := filepath.Join(tmpDir, "pkg.conf")
		target := filepath.Join(tmpDir, "user.conf")
		os.WriteFile(src, []byte(pkgContent), 0644)
		os.WriteFile(target, []byte(userContent), 0644)

		baseHash := eng.differ.CalculateStringHash(baseContent)
		eng.stateMgr.Set("testpkg", target, FileState{Path: target, Hash: baseHash})

		// Mock Resolver
		mockResolver := eng.resolver.(*MockConflictResolver)
		mockResolver.On("Resolve", target, mock.Anything).Return(ActionKeepUser, nil)

		err := eng.Apply("testpkg", src, target)
		require.NoError(t, err)

		// Verify User Kept
		finalContent, _ := os.ReadFile(target)
		assert.Equal(t, userContent, string(finalContent))

		mockResolver.AssertExpectations(t)
	})

	// Scenario 6: Conflict -> Interactive (Mocked: Update/Overwrite)
	t.Run("Conflict_Overwrite", func(t *testing.T) {
		eng, tmpDir, _ := setupEngineTest(t)
		defer os.RemoveAll(tmpDir)

		baseContent := "base"
		userContent := "user change"
		pkgContent := "pkg change"

		src := filepath.Join(tmpDir, "pkg.conf")
		target := filepath.Join(tmpDir, "user.conf")
		os.WriteFile(src, []byte(pkgContent), 0644)
		os.WriteFile(target, []byte(userContent), 0644)

		baseHash := eng.differ.CalculateStringHash(baseContent)
		eng.stateMgr.Set("testpkg", target, FileState{Path: target, Hash: baseHash})

		mockResolver := eng.resolver.(*MockConflictResolver)
		mockResolver.On("Resolve", target, mock.Anything).Return(ActionUpdate, nil)

		err := eng.Apply("testpkg", src, target)
		require.NoError(t, err)

		// Verify Overwritten
		finalContent, _ := os.ReadFile(target)
		assert.Equal(t, pkgContent, string(finalContent))

		// Verify Backup Created! (Implicit in engine logic for overwrites)
		// We can check if a .bak file exists
		matches, _ := filepath.Glob(target + ".*.bak")
		assert.Greater(t, len(matches), 0, "Backup should be created on overwrite conflict")
	})
}
