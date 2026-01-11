package mirror

import "time"

// MirrorBackend defines the interface for mirror management
type MirrorBackend interface {
	Name() string
	List() ([]Mirror, error)
	Test() ([]Mirror, error)
	Select(topN int, countries []string, sort string) error
	IsAvailable() bool
}

// Mirror represents a package mirror
type Mirror struct {
	URL     string
	Country string
	Speed   time.Duration
}
