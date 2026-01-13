package core

// OfficialBackend is the distribution-specific package manager interface.
// It composes fine-grained capabilities into a single interface.
type OfficialBackend interface {
	Backend
	PackageManager
	Searchable
	Informer
	Upgradable
	LocalInstaller
	FileProvider
	Exporter
	SecurityScanner
	Differ
}
