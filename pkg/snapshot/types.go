package snapshot

import (
	"context"
	"time"
)

const (
	// Remote Strategies
	StrategyRestic = "restic"
	StrategyRclone = "rclone"
	StrategyLocal  = "local" // For disk/USB targets

	// Backends
	BackendSnapper   = "snapper"
	BackendTimeshift = "timeshift"
	BackendRsync     = "rsync"
	BackendRestic    = "restic" // When list returns remote snapshots
)

// Snapshot represents a single system snapshot
type Snapshot struct {
	ID          string // Unique ID (e.g., timestamp or subvol ID)
	Timestamp   time.Time
	Description string
	Type        string   // "pre", "post", "manual", "scheduled"
	Path        string   // Local path (e.g. /.snapshots/1)
	Backend     string   // "snapper", "timeshift", "rsync"
	Size        int64    // Size in bytes
	Tags        []string // "cloud-synced", "encrypted"
}

// CreateOptions options for creating a snapshot
type CreateOptions struct {
	IncludeHome   bool
	Type          string
	Tags          []string
	TargetConfigs []string // List of configs/subvolumes to snapshot
	DryRun        bool     // Preview without executing
}

// ListOptions options for listing snapshots
type ListOptions struct {
	Remote bool
	Target *RemoteTarget // Specific target to list from (e.g. disk)
}

// RestoreOptions options for restoring
type RestoreOptions struct {
	PackagesOnly bool
	ConfigsOnly  bool
	HomeOnly     bool
	Force        bool // Bypass safety checks (e.g. for full system restore)
	DryRun       bool // Preview without executing
}

// PruneOptions options for pruning
type PruneOptions struct {
	OlderThan     time.Duration
	KeepLast      int
	KeepScheduled int
}

// DiffResult holds the result of diffing two snapshots.
type DiffResult struct {
	Added    []string
	Removed  []string
	Modified []string
}

// RemoteTarget represents a remote storage location
type RemoteTarget struct {
	Name string // "gdrive", "r2", "s3", "usb"
	Type string // "rclone", "ssh", "local"
	Path string // specific path or bucket
}

// RemoteOptions options for remote operations
type RemoteOptions struct {
	Compress  bool
	Bandwidth int // KB/s
	Delete    bool
	DryRun    bool // Preview without executing
}

// Manager defines the core snapshot operations
type Manager interface {
	// CRUD
	Create(ctx context.Context, desc string, opts CreateOptions) (*Snapshot, error)
	List(ctx context.Context, opts ListOptions) ([]Snapshot, error)
	Delete(ctx context.Context, id string) error
	Restore(ctx context.Context, id string, opts RestoreOptions) error
	Prune(ctx context.Context, opts PruneOptions) error

	// Remote Capabilities
	Push(ctx context.Context, id string, target RemoteTarget, opts RemoteOptions) error
	Pull(ctx context.Context, id string, source RemoteTarget, opts RemoteOptions) error

	// Comparison
	Diff(ctx context.Context, id1, id2 string) (DiffResult, error)

	// Backend Info
	GetBackendName() string
}

// ScheduleStatus represents the status of the scheduler
type ScheduleStatus struct {
	Enabled   bool
	NextRun   time.Time
	LastRun   time.Time
	Frequency string
}

// Scheduler defines automation operations
type Scheduler interface {
	Enable() error
	Disable() error
	Status() (ScheduleStatus, error)
	RunNow() error
}

// Key represents an encryption key
type Key struct {
	ID          string
	Description string
	Created     time.Time
	Fingerprint string
}

// KeyManager defines encryption key operations
type KeyManager interface {
	Generate(desc string) (string, error)
	Export(id string, path string) error
	Import(path string) error
	List() ([]Key, error)
	Delete(id string) error
}
