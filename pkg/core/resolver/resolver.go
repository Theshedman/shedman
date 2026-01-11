package resolver

import (
	"errors"
	"fmt"
	"strings"

	"github.com/theshedman/shedman/pkg/core/pkgdb"
)

// Errors for resolution
var (
	ErrPackageNotFound = errors.New("package not found")
	ErrVersionMismatch = errors.New("version constraint not satisfied")
)

// Version constraint operators
const (
	OpEqual        = "="
	OpGreaterEqual = ">="
	OpLessEqual    = "<="
	OpGreater      = ">"
	OpLess         = "<"
)

// Request represents a package install request
type Request struct {
	Name     string
	Version  string
	Operator string // =, >=, <=, >, <
	Source   string
	IsGroup  bool
}

// ParseRequest parses a package request string like "neovim@0.10.0", "neovim>=0.9.0", or "@dev"
func ParseRequest(input string) Request {
	req := Request{}

	// Check if it's a group
	if strings.HasPrefix(input, "@") {
		req.Name = input
		req.IsGroup = true
		return req
	}

	// Check for version constraint operators (order matters: >= before >)
	operators := []string{">=", "<=", ">", "<", "=", "@"}
	for _, op := range operators {
		if idx := strings.Index(input, op); idx != -1 {
			req.Name = input[:idx]
			req.Version = input[idx+len(op):]
			if op == "@" {
				req.Operator = OpEqual // @ is shorthand for exact version
			} else {
				req.Operator = op
			}
			return req
		}
	}

	// No version constraint
	req.Name = input
	return req
}

// Result holds the resolution result
type Result struct {
	ToInstall []pkgdb.PackageInfo
	ToUpgrade []pkgdb.PackageInfo
	ToRemove  []pkgdb.PackageInfo
	Conflicts []string
}

// Resolver resolves package dependencies
type Resolver struct {
	db pkgdb.PackageDB
}

// New creates a new Resolver
func New(db pkgdb.PackageDB) *Resolver {
	return &Resolver{db: db}
}

// Resolve resolves all packages and their dependencies
func (r *Resolver) Resolve(packages []string) (*Result, error) {
	result := &Result{}
	visited := make(map[string]bool)

	for _, pkg := range packages {
		req := ParseRequest(pkg)
		// For explicitly requested packages, require them to exist
		if err := r.resolveRequestedPackage(req, result, visited); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// resolveRequestedPackage resolves a user-requested package (must exist)
func (r *Resolver) resolveRequestedPackage(req Request, result *Result, visited map[string]bool) error {
	if visited[req.Name] {
		return nil
	}

	info, err := r.db.GetInfo(req.Name)
	if err != nil {
		return err
	}
	if info == nil {
		return fmt.Errorf("%w: %s", ErrPackageNotFound, req.Name)
	}

	// Validate version constraint if specified
	if req.Version != "" && req.Operator != "" {
		if !RequestMatchesPackage(req, info.Version) {
			return fmt.Errorf("%w: %s (have %s, need %s%s)",
				ErrVersionMismatch, req.Name, info.Version, req.Operator, req.Version)
		}
	}

	visited[req.Name] = true

	// Resolve dependencies (these can be optional)
	for _, dep := range info.Depends {
		if err := r.resolveDependency(dep, result, visited); err != nil {
			return err
		}
	}

	result.ToInstall = append(result.ToInstall, *info)
	return nil
}

// resolveDependency recursively resolves package dependencies (missing deps are skipped)
func (r *Resolver) resolveDependency(name string, result *Result, visited map[string]bool) error {
	if visited[name] {
		return nil
	}
	visited[name] = true

	info, err := r.db.GetInfo(name)
	if err != nil {
		return err
	}
	if info == nil {
		return nil // Dependency not found - may be provided elsewhere or already installed
	}

	// Resolve nested dependencies first
	for _, dep := range info.Depends {
		if err := r.resolveDependency(dep, result, visited); err != nil {
			return err
		}
	}

	// Add this package
	result.ToInstall = append(result.ToInstall, *info)
	return nil
}
