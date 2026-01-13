package core

// Package source constants
const (
	SourceOfficial = "official"
	SourceAUR      = "aur"
	SourceShedOS   = "shedos"
)

// Package type constants
const (
	PackageTypeArch = "arch" // Native Arch .pkg.tar.zst
)

// InstallReason denotes the reason a package was installed.
type InstallReason int

// InstallReason constants matches libalpm logic.
const (
	InstallReasonExplicit   InstallReason = 0
	InstallReasonDependency InstallReason = 1
)

// PackageInfo holds metadata about a package
type PackageInfo struct {
	Name          string
	Version       string
	Description   string
	Source        string // SourceOfficial, SourceAUR, SourceShedOS
	PackageType   string // PackageTypeArch or PackageTypeShed (for ShedOS packages)
	Depends       []string
	OptDepends    []string
	Provides      []string
	Conflicts     []string
	Replaces      []string // Packages this package replaces
	Size          int64
	InstalledSize int64
	DownloadURL   string // Direct download URL for ShedOS packages
	Checksum      string // SHA256 checksum for verification
	Signature     string // GPG signature or signature URL
}

// IsNativeArch returns true if this is a native Arch package
func (p *PackageInfo) IsNativeArch() bool {
	return p.PackageType == PackageTypeArch || p.PackageType == ""
}

// PackageDB is the interface for querying package database
type PackageDB interface {
	Search(query string) ([]PackageInfo, error)
	GetInfo(name string) (*PackageInfo, error)
}
