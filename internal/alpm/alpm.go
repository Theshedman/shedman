// Package pacman provides libalpm integration for the pacman backend.
package alpm

import (
	"fmt"
	"strings"

	"github.com/Jguer/go-alpm/v2"
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
	// IsIgnored returns true if the package is in the IgnorePkg list
	IsIgnored(pkgName string) bool
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
	handle     *alpm.Handle
	pacmanConf *PacmanConf
	ignorePkgs map[string]bool
}

// NewRealAlpmHandle creates a new libalpm handle using /etc/pacman.conf.
func NewRealAlpmHandle() (*RealAlpmHandle, error) {
	return NewRealAlpmHandleWithConfPath(DefaultPacmanConfPath)
}

// NewRealAlpmHandleWithConfPath creates a libalpm handle from a pacman.conf path.
func NewRealAlpmHandleWithConfPath(confPath string) (*RealAlpmHandle, error) {
	conf, err := ParsePacmanConf(confPath)
	if err != nil {
		// Fall back to defaults if parsing fails
		conf = DefaultPacmanConf()
	}
	return NewRealAlpmHandleWithConf(conf)
}

// NewRealAlpmHandleWithConf creates a libalpm handle from a PacmanConf.
func NewRealAlpmHandleWithConf(conf *PacmanConf) (*RealAlpmHandle, error) {
	h, err := alpm.Initialize(conf.RootDir, conf.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize libalpm: %w", err)
	}

	// Build ignore packages map
	ignorePkgs := make(map[string]bool)
	for _, pkg := range conf.IgnorePkg {
		ignorePkgs[pkg] = true
	}

	handle := &RealAlpmHandle{
		handle:     h,
		pacmanConf: conf,
		ignorePkgs: ignorePkgs,
	}

	// Register sync databases from pacman.conf
	if err := handle.registerSyncDatabases(); err != nil {
		_ = h.Release()

		return nil, fmt.Errorf("failed to register sync databases: %w", err)
	}

	return handle, nil
}

// NewRealAlpmHandleWithPaths creates a new libalpm handle with custom paths (legacy).
func NewRealAlpmHandleWithPaths(root, dbPath string) (*RealAlpmHandle, error) {
	conf := &PacmanConf{
		RootDir:      root,
		DBPath:       dbPath,
		SigLevel:     "Required DatabaseOptional",
		Repositories: DefaultPacmanConf().Repositories,
	}
	return NewRealAlpmHandleWithConf(conf)
}

// registerSyncDatabases registers all repositories from pacman.conf
func (r *RealAlpmHandle) registerSyncDatabases() error {
	siglevel := parseSigLevel(r.pacmanConf.SigLevel)

	for _, repo := range r.pacmanConf.Repositories {
		db, err := r.handle.RegisterSyncDB(repo.Name, siglevel)
		if err != nil {
			return fmt.Errorf("failed to register %s: %w", repo.Name, err)
		}

		// Set mirrors for the database
		if len(repo.Servers) > 0 {
			var servers []string
			for _, server := range repo.Servers {
				servers = append(servers, r.pacmanConf.ExpandVariables(server, repo.Name))
			}
			db.SetServers(servers)
		}
	}

	return nil
}

// parseSigLevel converts a SigLevel string to alpm.SigLevel
func parseSigLevel(level string) alpm.SigLevel {
	var sig alpm.SigLevel
	level = strings.ToLower(level)

	if strings.Contains(level, "required") {
		sig |= alpm.SigPackage
		sig |= alpm.SigDatabase
	}
	if strings.Contains(level, "optional") {
		sig |= alpm.SigPackageOptional
	}
	if strings.Contains(level, "databaseoptional") {
		sig |= alpm.SigDatabaseOptional
	}
	if strings.Contains(level, "trustall") || strings.Contains(level, "marginalok") {
		sig |= alpm.SigPackageMarginalOk
		sig |= alpm.SigDatabaseMarginalOk
	}

	return sig
}

// IsIgnored returns true if the package is in the IgnorePkg list
func (r *RealAlpmHandle) IsIgnored(pkgName string) bool {
	return r.ignorePkgs[pkgName]
}

// GetPacmanConf returns the parsed pacman.conf
func (r *RealAlpmHandle) GetPacmanConf() *PacmanConf {
	return r.pacmanConf
}

// SetupCallbacks configures libalpm callbacks for logging and questions.
// If autoConfirm is true, all questions will be auto-answered with yes.
func (r *RealAlpmHandle) SetupCallbacks(autoConfirm bool) {
	// Set question callback for NoConfirm behavior
	if autoConfirm {
		r.handle.SetQuestionCallback(func(_ interface{}, q alpm.QuestionAny) {
			// Auto-answer all questions with "yes"
			q.SetAnswer(true)
		}, nil)
	}

	// Set log callback for transaction progress
	r.handle.SetLogCallback(func(_ interface{}, lvl alpm.LogLevel, msg string) {
		if lvl <= alpm.LogWarning {
			// Only print warnings and errors
			_, _ = fmt.Printf("[%s] %s", logLevelString(lvl), msg)

		}
	}, nil)
}

// logLevelString returns a string representation of the log level
func logLevelString(lvl alpm.LogLevel) string {
	switch lvl {
	case alpm.LogError:
		return "ERROR"
	case alpm.LogWarning:
		return "WARN"
	case alpm.LogDebug:
		return "DEBUG"
	case alpm.LogFunction:
		return "FUNC"
	default:
		return "INFO"
	}
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
	_ = l.ForEach(func(db AlpmDB) error {

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
	_ = l.ForEach(func(pkg AlpmPackage) error {

		result = append(result, pkg)
		return nil
	})
	return result
}

// Len returns the number of packages.
func (l *RealAlpmPackageList) Len() int {
	count := 0
	_ = l.ForEach(func(pkg AlpmPackage) error {

		count++
		return nil
	})
	return count
}

// RealAlpmDependList wraps a real dependency list.
type RealAlpmDependList struct {
	list alpm.IDependList
}

// Slice returns dependencies as strings.
func (l *RealAlpmDependList) Slice() []string {
	var result []string
	_ = l.list.ForEach(func(dep *alpm.Depend) error {

		result = append(result, dep.Name)
		return nil
	})
	return result
}
