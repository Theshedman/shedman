package resolver

import (
	"errors"
	"fmt"
	"strings"

	"github.com/theshedman/shedman/pkg/core/pkgdb"
)

// Errors for dependency resolution
var (
	ErrCircularDependency = errors.New("circular dependency detected")
	ErrDepNotFound        = errors.New("dependency not found")
	ErrDepVersionMismatch = errors.New("dependency version constraint not satisfied")
)

// IsCircularDependency checks if an error is a circular dependency error
func IsCircularDependency(err error) bool {
	return errors.Is(err, ErrCircularDependency)
}

// OptionalDep represents an optional dependency with description
type OptionalDep struct {
	Name        string
	Description string
}

// ParseOptionalDep parses "python: python support" into name and description
func ParseOptionalDep(optDep string) OptionalDep {
	parts := strings.SplitN(optDep, ":", 2)
	od := OptionalDep{Name: strings.TrimSpace(parts[0])}
	if len(parts) > 1 {
		od.Description = strings.TrimSpace(parts[1])
	}
	return od
}

// DependencyNode represents a package in the dependency tree
type DependencyNode struct {
	Package          pkgdb.PackageInfo
	Children         []string      // Required dependencies
	OptionalDeps     []OptionalDep // Optional dependencies
	Depth            int           // Depth in the tree (0 = root)
	AlreadyInstalled bool          // Package is already installed on system
	Replaces         []string      // Packages this will replace
}

// DependencyTreeOptions configures dependency resolution behavior
type DependencyTreeOptions struct {
	SkipInstalled bool // Skip packages already installed
	StrictDeps    bool // Error on missing dependencies
	MaxDepth      int  // Maximum resolution depth (0 = unlimited)
}

// DefaultDependencyTreeOptions returns sensible defaults
func DefaultDependencyTreeOptions() DependencyTreeOptions {
	return DependencyTreeOptions{
		SkipInstalled: true,
		StrictDeps:    false,
		MaxDepth:      100,
	}
}

// DependencyTree builds and manages package dependency trees
type DependencyTree struct {
	db          pkgdb.PackageDB
	installedDB pkgdb.PackageDB // For checking installed packages
	nodes       map[string]*DependencyNode
	toRemove    map[string]bool // Packages to be replaced
	visiting    map[string]bool // For detecting circular dependencies
	visited     map[string]bool
	rootPkgs    []string
	depChain    []string // Current dependency chain for error messages
	opts        DependencyTreeOptions
}

// NewDependencyTree creates a new DependencyTree
func NewDependencyTree(db pkgdb.PackageDB) *DependencyTree {
	return NewDependencyTreeWithOptions(db, nil, DefaultDependencyTreeOptions())
}

// NewDependencyTreeWithOptions creates a DependencyTree with custom options
func NewDependencyTreeWithOptions(db, installedDB pkgdb.PackageDB, opts DependencyTreeOptions) *DependencyTree {
	return &DependencyTree{
		db:          db,
		installedDB: installedDB,
		nodes:       make(map[string]*DependencyNode),
		toRemove:    make(map[string]bool),
		visiting:    make(map[string]bool),
		visited:     make(map[string]bool),
		depChain:    make([]string, 0),
		opts:        opts,
	}
}

// Build builds the dependency tree for the given packages
func (dt *DependencyTree) Build(packages []string) error {
	dt.rootPkgs = packages
	for _, pkg := range packages {
		if err := dt.buildNode(pkg, 0); err != nil {
			return err
		}
	}
	return nil
}

// buildNode recursively builds a node and its dependencies
func (dt *DependencyTree) buildNode(name string, depth int) error {
	// Check max depth
	if dt.opts.MaxDepth > 0 && depth > dt.opts.MaxDepth {
		return fmt.Errorf("dependency resolution exceeded max depth (%d)", dt.opts.MaxDepth)
	}

	// Parse version constraint from dependency string
	depReq := parseDependencyString(name)
	queriedName := depReq.Name

	// Check if already installed on system
	var isInstalled bool
	if dt.installedDB != nil {
		installedInfo, _ := dt.installedDB.GetInfo(queriedName)
		if installedInfo != nil {
			isInstalled = true
			// If skip installed is enabled and it's not a root package, skip
			if dt.opts.SkipInstalled && depth > 0 {
				dt.visited[queriedName] = true
				return nil
			}
		}
	}

	// Look up the package (may return a different package via provides)
	info, err := dt.db.GetInfo(queriedName)
	if err != nil {
		return err
	}
	if info == nil {
		// Dependency not found
		if dt.opts.StrictDeps {
			return fmt.Errorf("%w: %s", ErrDepNotFound, queriedName)
		}
		// Might be provided by another package or already installed
		return nil
	}

	// Use the actual package name (may differ from queried name if provides)
	actualName := info.Name

	// Already fully processed
	if dt.visited[actualName] {
		return nil
	}

	// Circular dependency detection
	if dt.visiting[actualName] {
		return fmt.Errorf("%w: %s (chain: %s → %s)",
			ErrCircularDependency, actualName, strings.Join(dt.depChain, " → "), actualName)
	}

	dt.visiting[actualName] = true
	dt.depChain = append(dt.depChain, actualName)
	defer func() {
		dt.visiting[actualName] = false
		dt.depChain = dt.depChain[:len(dt.depChain)-1]
	}()

	// Validate version constraint if specified
	if depReq.Version != "" && depReq.Operator != "" {
		if !MatchesVersionConstraint(info.Version, depReq.Version, depReq.Operator) {
			return fmt.Errorf("%w: %s (have %s, need %s%s)",
				ErrDepVersionMismatch, actualName, info.Version, depReq.Operator, depReq.Version)
		}
	}

	// Parse optional dependencies
	optDeps := make([]OptionalDep, 0, len(info.OptDepends))
	for _, od := range info.OptDepends {
		optDeps = append(optDeps, ParseOptionalDep(od))
	}

	// Handle replaces - mark old packages for removal
	var replaces []string
	if len(info.Replaces) > 0 {
		replaces = info.Replaces
		for _, oldPkg := range info.Replaces {
			dt.toRemove[oldPkg] = true
		}
	}

	node := &DependencyNode{
		Package:          *info,
		Children:         extractDepNames(info.Depends),
		OptionalDeps:     optDeps,
		Depth:            depth,
		AlreadyInstalled: isInstalled,
		Replaces:         replaces,
	}
	dt.nodes[actualName] = node

	// Recursively build dependencies
	for _, dep := range info.Depends {
		if err := dt.buildNode(dep, depth+1); err != nil {
			return err
		}
	}
	dt.visited[actualName] = true
	return nil
}

// extractDepNames removes version constraints from dependency list
func extractDepNames(deps []string) []string {
	names := make([]string, len(deps))
	for i, dep := range deps {
		names[i] = parseDependencyString(dep).Name
	}
	return names
}

// parseDependencyString parses "lib>=2.0" into Request with Name, Version, Operator
func parseDependencyString(dep string) Request {
	// Use ParseRequest for consistent parsing
	return ParseRequest(dep)
}

// GetNodes returns all nodes in the tree
func (dt *DependencyTree) GetNodes() map[string]*DependencyNode {
	return dt.nodes
}

// GetDependencies returns the direct dependencies of a package
func (dt *DependencyTree) GetDependencies(name string) []string {
	if node, ok := dt.nodes[name]; ok {
		return node.Children
	}
	return nil
}

// GetOptionalDependencies returns the optional dependencies of a package
func (dt *DependencyTree) GetOptionalDependencies(name string) []OptionalDep {
	if node, ok := dt.nodes[name]; ok {
		return node.OptionalDeps
	}
	return nil
}

// GetInstallOrder returns packages in topological order (dependencies first)
func (dt *DependencyTree) GetInstallOrder() []string {
	order := make([]string, 0, len(dt.nodes))
	visited := make(map[string]bool)

	var visit func(name string)
	visit = func(name string) {
		if visited[name] {
			return
		}
		visited[name] = true

		node, ok := dt.nodes[name]
		if !ok {
			return
		}

		// Visit dependencies first
		for _, dep := range node.Children {
			visit(dep)
		}

		order = append(order, name)
	}

	// Start from root packages
	for _, root := range dt.rootPkgs {
		visit(root)
	}

	// Also include any nodes not reachable from roots
	for name := range dt.nodes {
		visit(name)
	}

	return order
}

// GetTotalSize returns the total download size of all packages
func (dt *DependencyTree) GetTotalSize() int64 {
	var total int64
	for _, node := range dt.nodes {
		total += node.Package.Size
	}
	return total
}

// GetTotalInstalledSize returns the total installed size of all packages
func (dt *DependencyTree) GetTotalInstalledSize() int64 {
	var total int64
	for _, node := range dt.nodes {
		total += node.Package.InstalledSize
	}
	return total
}

// HasOptionalDeps returns true if any package has optional dependencies
func (dt *DependencyTree) HasOptionalDeps() bool {
	for _, node := range dt.nodes {
		if len(node.OptionalDeps) > 0 {
			return true
		}
	}
	return false
}

// GetAllOptionalDeps returns all optional dependencies across all packages
func (dt *DependencyTree) GetAllOptionalDeps() map[string][]OptionalDep {
	result := make(map[string][]OptionalDep)
	for name, node := range dt.nodes {
		if len(node.OptionalDeps) > 0 {
			result[name] = node.OptionalDeps
		}
	}
	return result
}

// GetPackagesToRemove returns packages that will be replaced by new packages
func (dt *DependencyTree) GetPackagesToRemove() []string {
	result := make([]string, 0, len(dt.toRemove))
	for pkg := range dt.toRemove {
		result = append(result, pkg)
	}
	return result
}

// GetNewPackages returns packages that are not already installed
func (dt *DependencyTree) GetNewPackages() []pkgdb.PackageInfo {
	result := make([]pkgdb.PackageInfo, 0)
	for _, node := range dt.nodes {
		if !node.AlreadyInstalled {
			result = append(result, node.Package)
		}
	}
	return result
}
