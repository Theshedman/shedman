package snapshot

// SnapshotBackend interface for different snapshot implementations (btrfs, rsync, timeshift)
type SnapshotBackend interface {
	// Name returns the backend name
	Name() string

	// Create creates a new snapshot
	Create(opts CreateOptions) (*Snapshot, error)

	// Restore restores a snapshot
	Restore(id string, opts RestoreOptions) error

	// List returns all available snapshots
	List() ([]*Snapshot, error)

	// Delete deletes a snapshot
	Delete(id string) error

	// Get returns a specific snapshot
	Get(id string) (*Snapshot, error)

	// IsAvailable checks if the backend is usable (e.g. btrfs volume exists)
	IsAvailable() bool
}
