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
	stateMgr  StateManager
	backupMgr BackupManager
	differ    *Differ
	resolver  ConflictResolver
}

// NewConfigEngine creates a new config engine
func NewConfigEngine(state StateManager, backup BackupManager, differ *Differ, resolver ConflictResolver) *ConfigEngine {
	return &ConfigEngine{
		stateMgr:  state,
		backupMgr: backup,
		differ:    differ,
		resolver:  resolver,
	}
}

// Apply applies a single configuration file from source to target
func (e *ConfigEngine) Apply(packageName, sourcePath, targetPath string) error {
	// 1. Load State

	// 2. Calculate Hashes
	// New (Incoming Package)
	newHash, err := e.differ.CalculateHash(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to hash source: %w", err)
	}

	// Current (User's File)
	var currentHash string
	targetExists := false
	if _, err := os.Stat(targetPath); err == nil {
		targetExists = true
		currentHash, err = e.differ.CalculateHash(targetPath)
		if err != nil {
			return fmt.Errorf("failed to hash target: %w", err)
		}
	}

	// Base (Last Applied)
	state, hasState := e.stateMgr.Get(packageName, targetPath)
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

		resolution, err := e.resolver.Resolve(targetPath, diff)
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
	if _, err := e.backupMgr.Backup(targetPath); err != nil {
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
	e.stateMgr.Set(pkg, path, FileState{
		Path:         path,
		Hash:         hash,
		LastModified: time.Now(),
		Version:      "latest",
	})
	e.stateMgr.Save()
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
	userContent, err := e.differ.ReadFile(userPath)
	if err != nil {
		return "", err
	}
	pkgContent, err := e.differ.ReadFile(pkgPath)
	if err != nil {
		return "", err
	}
	return e.differ.GenerateDiff(userPath, userContent, pkgPath, pkgContent)
}
