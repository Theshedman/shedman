// Package pacman provides libalpm integration for the pacman backend.
package pacman

import (
	"fmt"

	"github.com/Jguer/go-alpm/v2"
	"github.com/theshedman/shedman/pkg/shedman/pkgdb"
)

// AlpmHandle abstracts libalpm operations for testability.
// This interface wraps the real go-alpm Handle to enable mocking in tests.
type AlpmHandle interface {
	// LocalDb returns the local package database
	LocalDb() AlpmDB
	// SyncDbs returns the list of sync databases
	SyncDbs() AlpmDBList
	// Release frees the libalpm handle
	Release() error
	// Root returns the root path
	Root() string
	// DBPath returns the database path
	DBPath() string

	// Transaction methods for Install/Remove/Upgrade
	// TransInit initializes a transaction with the given flags
	TransInit(flags alpm.TransFlag) error
	// TransPrepare prepares the transaction
	TransPrepare() error
	// TransCommit commits the transaction
	TransCommit() error
	// TransRelease releases the transaction
	TransRelease() error
	// AddPkg adds a package to the transaction (for install)
	AddPkg(pkg AlpmPackage) error
	// RemovePkg adds a package for removal
	RemovePkg(pkg AlpmPackage) error
	// SyncDBsForce returns sync databases and registers them if needed
	SyncDBsForce() AlpmDBList
	// GetRawHandle returns the underlying alpm.Handle for advanced operations
	GetRawHandle() *alpm.Handle
}

// AlpmDB abstracts a libalpm database (local or sync).
type AlpmDB interface {
	// Name returns the database name
	Name() string
	// Pkg returns a package by name, or nil if not found
	Pkg(name string) AlpmPackage
	// PkgCache returns all packages in this database
	PkgCache() AlpmPackageList
	// Search searches for packages matching targets
	Search(targets []string) AlpmPackageList
}

// AlpmDBList abstracts a list of databases.
type AlpmDBList interface {
	// ForEach iterates over each database
	ForEach(func(AlpmDB) error) error
	// Slice returns all databases as a slice
	Slice() []AlpmDB
}

// AlpmPackage abstracts a libalpm package.
type AlpmPackage interface {
	// Name returns the package name
	Name() string
	// Version returns the package version
	Version() string
	// Description returns the package description
	Description() string
	// Depends returns the list of dependencies
	Depends() AlpmDependList
	// OptionalDepends returns the list of optional dependencies
	OptionalDepends() AlpmDependList
	// Provides returns the list of provided packages
	Provides() AlpmDependList
	// Conflicts returns the list of conflicting packages
	Conflicts() AlpmDependList
	// Size returns the installed size in bytes
	Size() int64
	// ISize returns the installed size in bytes
	ISize() int64
	// Files returns the list of files owned by this package
	Files() []alpm.File
	// DB returns the database this package belongs to
	DB() AlpmDB
}

// AlpmPackageList abstracts a list of packages.
type AlpmPackageList interface {
	// ForEach iterates over each package
	ForEach(func(AlpmPackage) error) error
	// Slice returns all packages as a slice
	Slice() []AlpmPackage
	// Len returns the number of packages
	Len() int
}

// AlpmDependList abstracts a list of dependencies.
type AlpmDependList interface {
	// Slice returns all dependencies as strings
	Slice() []string
}

// RealAlpmHandle wraps the actual go-alpm Handle.
type RealAlpmHandle struct {
	handle *alpm.Handle
}

// NewRealAlpmHandle creates a new libalpm handle with default paths.
func NewRealAlpmHandle() (*RealAlpmHandle, error) {
	return NewRealAlpmHandleWithPaths("/", "/var/lib/pacman")
}

// NewRealAlpmHandleWithPaths creates a new libalpm handle with custom paths.
func NewRealAlpmHandleWithPaths(root, dbPath string) (*RealAlpmHandle, error) {
	h, err := alpm.Initialize(root, dbPath)
	if err != nil {
		return nil, err
	}
	return &RealAlpmHandle{handle: h}, nil
}

// LocalDb returns the local database.
func (r *RealAlpmHandle) LocalDb() AlpmDB {
	db, err := r.handle.LocalDB()
	if err != nil {
		return nil
	}
	return &RealAlpmDB{db: db.(*alpm.DB)}
}

// SyncDbs returns the list of sync databases.
func (r *RealAlpmHandle) SyncDbs() AlpmDBList {
	list, err := r.handle.SyncDBs()
	if err != nil {
		return &RealAlpmDBList{list: nil}
	}
	return &RealAlpmDBList{list: list}
}

// Release frees the handle.
func (r *RealAlpmHandle) Release() error {
	return r.handle.Release()
}

// Root returns the root path.
func (r *RealAlpmHandle) Root() string {
	root, _ := r.handle.Root()
	return root
}

// DBPath returns the database path.
func (r *RealAlpmHandle) DBPath() string {
	dbPath, _ := r.handle.DBPath()
	return dbPath
}

// TransInit initializes a transaction.
func (r *RealAlpmHandle) TransInit(flags alpm.TransFlag) error {
	return r.handle.TransInit(flags)
}

// TransPrepare prepares the transaction.
func (r *RealAlpmHandle) TransPrepare() error {
	return r.handle.TransPrepare()
}

// TransCommit commits the transaction.
func (r *RealAlpmHandle) TransCommit() error {
	return r.handle.TransCommit()
}

// TransRelease releases the transaction.
func (r *RealAlpmHandle) TransRelease() error {
	return r.handle.TransRelease()
}

// AddPkg adds a package to the transaction for installation.
func (r *RealAlpmHandle) AddPkg(pkg AlpmPackage) error {
	realPkg, ok := pkg.(*RealAlpmPackage)
	if !ok {
		return ErrInvalidPackageType
	}
	return r.handle.AddPkg(realPkg.pkg)
}

// RemovePkg adds a package for removal.
func (r *RealAlpmHandle) RemovePkg(pkg AlpmPackage) error {
	realPkg, ok := pkg.(*RealAlpmPackage)
	if !ok {
		return ErrInvalidPackageType
	}
	return r.handle.RemovePkg(realPkg.pkg)
}

// SyncDBsForce returns sync databases, registering them if needed.
func (r *RealAlpmHandle) SyncDBsForce() AlpmDBList {
	return r.SyncDbs()
}

// GetRawHandle returns the underlying alpm.Handle.
func (r *RealAlpmHandle) GetRawHandle() *alpm.Handle {
	return r.handle
}

// ErrInvalidPackageType is returned when a package type assertion fails.
var ErrInvalidPackageType = fmt.Errorf("invalid package type")

// RealAlpmDB wraps a real go-alpm DB.
type RealAlpmDB struct {
	db *alpm.DB
}

// Name returns the database name.
func (d *RealAlpmDB) Name() string {
	return d.db.Name()
}

// Pkg returns a package by name.
func (d *RealAlpmDB) Pkg(name string) AlpmPackage {
	pkg := d.db.Pkg(name)
	if pkg == nil {
		return nil
	}
	return &RealAlpmPackage{pkg: pkg, parentDB: d}
}

// PkgCache returns all packages.
func (d *RealAlpmDB) PkgCache() AlpmPackageList {
	return &RealAlpmPackageList{list: d.db.PkgCache(), parentDB: d}
}

// Search searches for packages.
func (d *RealAlpmDB) Search(targets []string) AlpmPackageList {
	return &RealAlpmPackageList{list: d.db.Search(targets), parentDB: d}
}

// RealAlpmDBList wraps a real go-alpm DBList.
type RealAlpmDBList struct {
	list alpm.IDBList
}

// ForEach iterates over databases.
func (l *RealAlpmDBList) ForEach(f func(AlpmDB) error) error {
	return l.list.ForEach(func(db alpm.IDB) error {
		return f(&RealAlpmDB{db: db.(*alpm.DB)})
	})
}

// Slice returns all databases.
func (l *RealAlpmDBList) Slice() []AlpmDB {
	var result []AlpmDB
	l.ForEach(func(db AlpmDB) error {
		result = append(result, db)
		return nil
	})
	return result
}

// RealAlpmPackage wraps a real go-alpm Package.
type RealAlpmPackage struct {
	pkg      alpm.IPackage
	parentDB *RealAlpmDB
}

// Name returns the package name.
func (p *RealAlpmPackage) Name() string {
	return p.pkg.Name()
}

// Version returns the package version.
func (p *RealAlpmPackage) Version() string {
	return p.pkg.Version()
}

// Description returns the package description.
func (p *RealAlpmPackage) Description() string {
	return p.pkg.Description()
}

// Depends returns dependencies.
func (p *RealAlpmPackage) Depends() AlpmDependList {
	return &RealAlpmDependList{list: p.pkg.Depends()}
}

// OptionalDepends returns optional dependencies.
func (p *RealAlpmPackage) OptionalDepends() AlpmDependList {
	return &RealAlpmDependList{list: p.pkg.OptionalDepends()}
}

// Provides returns provided packages.
func (p *RealAlpmPackage) Provides() AlpmDependList {
	return &RealAlpmDependList{list: p.pkg.Provides()}
}

// Conflicts returns conflicting packages.
func (p *RealAlpmPackage) Conflicts() AlpmDependList {
	return &RealAlpmDependList{list: p.pkg.Conflicts()}
}

// Size returns the download size.
func (p *RealAlpmPackage) Size() int64 {
	return p.pkg.Size()
}

// ISize returns the installed size.
func (p *RealAlpmPackage) ISize() int64 {
	return p.pkg.ISize()
}

// Files returns the package files.
func (p *RealAlpmPackage) Files() []alpm.File {
	return p.pkg.Files()
}

// DB returns the parent database.
func (p *RealAlpmPackage) DB() AlpmDB {
	return p.parentDB
}

// RealAlpmPackageList wraps a real package list.
type RealAlpmPackageList struct {
	list     alpm.IPackageList
	parentDB *RealAlpmDB
}

// ForEach iterates over packages.
func (l *RealAlpmPackageList) ForEach(f func(AlpmPackage) error) error {
	return l.list.ForEach(func(pkg alpm.IPackage) error {
		return f(&RealAlpmPackage{pkg: pkg, parentDB: l.parentDB})
	})
}

// Slice returns all packages.
func (l *RealAlpmPackageList) Slice() []AlpmPackage {
	var result []AlpmPackage
	l.ForEach(func(pkg AlpmPackage) error {
		result = append(result, pkg)
		return nil
	})
	return result
}

// Len returns the number of packages.
func (l *RealAlpmPackageList) Len() int {
	return len(l.Slice())
}

// RealAlpmDependList wraps a real dependency list.
type RealAlpmDependList struct {
	list alpm.IDependList
}

// Slice returns dependencies as strings.
func (l *RealAlpmDependList) Slice() []string {
	var result []string
	l.list.ForEach(func(dep *alpm.Depend) error {
		result = append(result, dep.Name)
		return nil
	})
	return result
}

// PackageToInfo converts an AlpmPackage to pkgdb.PackageInfo.
func PackageToInfo(pkg AlpmPackage) pkgdb.PackageInfo {
	info := pkgdb.PackageInfo{
		Name:          pkg.Name(),
		Version:       pkg.Version(),
		Description:   pkg.Description(),
		Source:        pkgdb.SourceOfficial,
		Depends:       pkg.Depends().Slice(),
		OptDepends:    pkg.OptionalDepends().Slice(),
		Provides:      pkg.Provides().Slice(),
		Conflicts:     pkg.Conflicts().Slice(),
		Size:          pkg.Size(),
		InstalledSize: pkg.ISize(),
	}
	return info
}
