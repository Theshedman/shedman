package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// ConfigEngine manages configuration application logic
type ConfigEngine struct {
	StateMgr  StateManager
	BackupMgr BackupManager
	Differ    *Differ
	Resolver  ConflictResolver
	provider  SourceProvider
}

// NewConfigEngine creates a new config engine
func NewConfigEngine(state StateManager, backup BackupManager, differ *Differ, resolver ConflictResolver, provider SourceProvider) *ConfigEngine {
	return &ConfigEngine{
		StateMgr:  state,
		BackupMgr: backup,
		Differ:    differ,
		Resolver:  resolver,
		provider:  provider,
	}
}

// Apply applies a single configuration file from source to target
func (e *ConfigEngine) Apply(packageName, sourcePath, targetPath string) error {
	// 1. Load State

	// 2. Calculate Hashes
	// New (Incoming Package)
	newHash, err := e.Differ.CalculateHash(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to hash source: %w", err)
	}

	// Current (User's File)
	var currentHash string
	targetExists := false
	if _, err := os.Stat(targetPath); err == nil {
		targetExists = true
		currentHash, err = e.Differ.CalculateHash(targetPath)
		if err != nil {
			return fmt.Errorf("failed to hash target: %w", err)
		}
	}

	// Base (Last Applied)
	state, hasState := e.StateMgr.Get(packageName, targetPath)
	var baseHash string
	if hasState {
		baseHash = state.Hash
	}

	// 3. Decision Matrix
	action := ActionNoop

	if !targetExists {
		// Scenario 1: Fresh Install
		action = ActionCopy
	} else if !hasState {
		// Target exists but no state (Untracked)
		if currentHash == newHash {
			action = ActionNoop
		} else {
			// Content differs from new package version
			action = ActionConflict
		}
	} else {
		// Full Three-Way Logic
		userChanged := currentHash != baseHash
		pkgChanged := newHash != baseHash

		if !userChanged && !pkgChanged {
			action = ActionNoop
		} else if userChanged && !pkgChanged {
			// User changed, pkg same.
			action = ActionKeepUser
		} else if !userChanged && pkgChanged {
			// User same, pkg changed.
			action = ActionUpdate
		} else if userChanged && pkgChanged {
			// Both changed.
			if currentHash == newHash {
				// Accidental convergence
				action = ActionNoop
			} else {
				action = ActionConflict
			}
		}
	}

	// 4. Execute Action
	switch action {
	case ActionUpdate:
		// Backup and overwrite
		if err := e.applyPackageVersion(packageName, sourcePath, targetPath, newHash); err != nil {
			return err
		}

	case ActionCopy:
		// Fresh install. Ensure dir exists.
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("mkdir failed: %w", err)
		}
		if err := e.copyFile(sourcePath, targetPath); err != nil {
			return fmt.Errorf("install copy failed: %w", err)
		}
		e.updateState(packageName, targetPath, newHash)

	case ActionKeepUser:
		// Do not modify file or update state hash to preserve divergence
		return nil

	case ActionNoop:
		// Update state if untracked or convergent
		if !hasState {
			e.updateState(packageName, targetPath, newHash)
		} else if baseHash != newHash && currentHash == newHash {
			e.updateState(packageName, targetPath, newHash)
		}
		return nil

	case ActionConflict:
		// Resolve using resolver
		diff, err := e.generateDisplayDiff(targetPath, sourcePath)
		if err != nil {
			return fmt.Errorf("diff gen failed: %w", err)
		}

		resolution, err := e.Resolver.Resolve(targetPath, diff)
		if err != nil {
			return fmt.Errorf("resolution failed: %w", err)
		}

		switch resolution {
		case ActionKeepUser:
			return nil
		case ActionUpdate, ActionReset:
			if err := e.applyPackageVersion(packageName, sourcePath, targetPath, newHash); err != nil {
				return err
			}
		default:
			// Noop
		}
	}

	return nil
}

// applyPackageVersion backs up the current file (if exists), overwrites it with source, and updates state
func (e *ConfigEngine) applyPackageVersion(packageName, sourcePath, targetPath, newHash string) error {
	if _, err := e.BackupMgr.Backup(targetPath); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}
	if err := e.copyFile(sourcePath, targetPath); err != nil {
		return fmt.Errorf("update copy failed: %w", err)
	}
	e.updateState(packageName, targetPath, newHash)
	return nil
}

// updateState updates the persistence layer
func (e *ConfigEngine) updateState(pkg, path, hash string) {
	e.StateMgr.Set(pkg, path, FileState{
		Path:         path,
		Hash:         hash,
		LastModified: time.Now(),
		Version:      "latest",
	})
	e.StateMgr.Save()
}

// copyFile copies content from src to dst atomically
func (e *ConfigEngine) copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	// Atomic write via temp file
	tmp := dst + ".tmp"
	out, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}

	return os.Rename(tmp, dst)
}

// generateDisplayDiff generates diff for interactive display
func (e *ConfigEngine) generateDisplayDiff(userPath, pkgPath string) (string, error) {
	userContent, err := e.Differ.ReadFile(userPath)
	if err != nil {
		return "", err
	}
	pkgContent, err := e.Differ.ReadFile(pkgPath)
	if err != nil {
		return "", err
	}
	return e.Differ.GenerateDiff(userPath, userContent, pkgPath, pkgContent)
}

// GetOriginal retrieves the original content of a tracked file.
func (e *ConfigEngine) GetOriginal(path string) ([]byte, error) {
	if e.provider == nil {
		return nil, fmt.Errorf("no source provider configured")
	}
	return e.provider.GetOriginalContent(path)
}

// GetFileOwner returns the owner package of the file.
func (e *ConfigEngine) GetFileOwner(path string) (string, error) {
	if e.provider == nil {
		return "", fmt.Errorf("no source provider configured")
	}
	return e.provider.GetOwner(path)
}
