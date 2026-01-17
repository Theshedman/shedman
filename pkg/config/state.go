package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/theshedman/shedman/internal/util"
)

// JSONStateManager implements StateManager using a JSON file
type JSONStateManager struct {
	path  string
	state ConfigState
	mu    sync.RWMutex
}

// NewJSONStateManager creates a new state manager
func NewJSONStateManager(path string) *JSONStateManager {
	return &JSONStateManager{
		path: path,
		state: ConfigState{
			Configs: make(map[string]map[string]FileState),
		},
	}
}

// Load loads the state from disk
func (m *JSONStateManager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.path)
	if err != nil {
		if os.IsNotExist(err) {
			// Initialize empty state if file doesn't exist
			m.state = ConfigState{
				Configs: make(map[string]map[string]FileState),
			}
			return nil
		}
		return fmt.Errorf("failed to read state file: %w", err)
	}

	if err := json.Unmarshal(data, &m.state); err != nil {
		return fmt.Errorf("failed to parse state file: %w", err)
	}

	// Ensure map is initialized if json was null
	if m.state.Configs == nil {
		m.state.Configs = make(map[string]map[string]FileState)
	}

	return nil
}

// Save saves the state to disk atomically
func (m *JSONStateManager) Save() error {
	m.mu.RLock()
	data, err := json.MarshalIndent(m.state, "", "  ")
	m.mu.RUnlock()

	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(m.path)
	if err := os.MkdirAll(dir, util.DirPermissions); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Atomic write: write to temp file then rename
	tmpFile := m.path + ".tmp"
	if err := os.WriteFile(tmpFile, data, util.FilePermissions); err != nil {
		return fmt.Errorf("failed to write tmporary state file: %w", err)
	}

	if err := os.Rename(tmpFile, m.path); err != nil {
		return fmt.Errorf("failed to rename state file: %w", err)
	}

	return nil
}

// Get retrieves state for a specific file
func (m *JSONStateManager) Get(packageName, relPath string) (*FileState, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pkgConfig, ok := m.state.Configs[packageName]
	if !ok {
		return nil, false
	}

	state, ok := pkgConfig[relPath]
	return &state, ok
}

// Set updates state for a specific file
func (m *JSONStateManager) Set(packageName, relPath string, state FileState) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state.Configs[packageName] == nil {
		m.state.Configs[packageName] = make(map[string]FileState)
	}

	m.state.Configs[packageName][relPath] = state
}

// List returns all tracked file states flattened
func (m *JSONStateManager) List() []FileState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []FileState
	for _, pkgMap := range m.state.Configs {
		for _, state := range pkgMap {
			list = append(list, state)
		}
	}
	return list
}
