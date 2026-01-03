package pkgdb

// Package source constants
const (
	SourceOfficial = "official"
	SourceAUR      = "aur"
	SourceShedOS   = "shedos"
)

// PackageInfo holds metadata about a package
type PackageInfo struct {
	Name          string
	Version       string
	Description   string
	Source        string
	Depends       []string
	OptDepends    []string
	Provides      []string
	Conflicts     []string
	Replaces      []string // Packages this package replaces
	Size          int64
	InstalledSize int64
}

// PackageDB is the interface for querying package database
type PackageDB interface {
	Search(query string) ([]PackageInfo, error)
	GetInfo(name string) (*PackageInfo, error)
}
