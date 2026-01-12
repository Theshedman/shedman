package config

import "time"

// FileState represents the tracked state of a single configuration file
type FileState struct {
	Path         string    `json:"path"`
	Hash         string    `json:"hash"` // SHA256 hash of the content
	Version      string    `json:"version"`
	LastModified time.Time `json:"last_modified"`
}

// ConfigState represents the persistent state of all tracked configurations
type ConfigState struct {
	Configs map[string]map[string]FileState `json:"configs"` // PackageName -> RelativeFilePath -> FileState
}

// StateManager handles loading and saving configuration state
type StateManager interface {
	Load() error
	Save() error
	Get(packageName, relPath string) (*FileState, bool)
	Set(packageName, relPath string, state FileState)
}

// BackupManager handles backing up user configuration files
type BackupManager interface {
	// Backup creates a timestamped backup of the file at path
	Backup(path string) (string, error)
	// Rotate keeps only the last N backups for a given file
	Rotate(path string, keep int) error
}

// Action represents the decision made by the engine
type Action int

const (
	ActionCopy     Action = iota // Copy new file (fresh install)
	ActionNoop                   // Do nothing (files match)
	ActionKeepUser               // Keep user version (user modified, package matched)
	ActionUpdate                 // Update to new version (user matched, package updated)
	ActionConflict               // Conflict (both modified)
	ActionReset                  // Reset to package version (user modified but opted to reset)
)

// String returns string representation of Action
func (a Action) String() string {
	switch a {
	case ActionCopy:
		return "COPY"
	case ActionNoop:
		return "NOOP"
	case ActionKeepUser:
		return "KEEP_USER"
	case ActionUpdate:
		return "UPDATE"
	case ActionConflict:
		return "CONFLICT"
	case ActionReset:
		return "RESET"
	default:
		return "UNKNOWN"
	}
}

// ConflictResolver handles interactive conflict resolution
type ConflictResolver interface {
	// Resolve prompts the user to resolve a conflict
	Resolve(file string, diff string) (Action, error)
}
