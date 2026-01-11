package snapshot

import "time"

// Snapshot represents a system snapshot
type Snapshot struct {
	ID          string    `json:"id"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
	Type        string    `json:"type"` // "pre", "post", "manual"
	Path        string    `json:"path"`
}

// CreateOptions holds options for creating a snapshot
type CreateOptions struct {
	Description string
	Type        string
	IncludeHome bool
}

// RestoreOptions holds options for restoring a snapshot
type RestoreOptions struct {
	Force  bool
	Backup bool // Create a backup of current state before restore
	Verify bool
}
