package alpm

// Package sources
const (
	SourceOfficial = "official"
	SourceAUR      = "aur"
	SourceShedOS   = "shedos"
)

// PackageInfo represents package information (internal version of core.PackageInfo)
type PackageInfo struct {
	Name          string
	Version       string
	Description   string
	URL           string
	Source        string
	Installed     bool
	Depends       []string
	OptDepends    []string
	Provides      []string
	Conflicts     []string
	Replaces      []string
	Size          int64
	InstalledSize int64
	Packager      string
	BuildDate     int64
	InstallDate   int64
	Reason        string
	Groups        []string
	Licenses      []string
	Architectures []string
}
