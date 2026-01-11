package core

type PackageBackend interface {
	Sync() error
	Name() string
}
