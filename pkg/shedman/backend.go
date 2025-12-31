package shedman

type PackageBackend interface {
	Sync() error
	Name() string
}