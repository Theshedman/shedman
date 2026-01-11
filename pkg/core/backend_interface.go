package core

// Backend is the minimum interface all backends must implement.
// It provides basic identification and availability checks.
type Backend interface {
	// Name returns the unique name of the backend (e.g., "pacman")
	Name() string

	// Sync refreshes the backend's internal database (e.g., pacman -Sy)
	Sync() error

	// IsAvailable checks if the backend's underlying tools are present
	IsAvailable() bool
}
