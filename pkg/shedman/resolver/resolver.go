package resolver

import (
	"strings"

	"github.com/theshedman/shedman/pkg/shedman/pkgdb"
)

// Request represents a package install request
type Request struct {
	Name    string
	Version string
	Source  string
	IsGroup bool
}

// ParseRequest parses a package request string like "neovim@0.10.0" or "@dev"
func ParseRequest(input string) Request {
	req := Request{}

	// Check if it's a group
	if strings.HasPrefix(input, "@") {
		req.Name = input
		req.IsGroup = true
		return req
	}

	// Check for version constraint
	if idx := strings.Index(input, "@"); idx != -1 {
		req.Name = input[:idx]
		req.Version = input[idx+1:]
	} else {
		req.Name = input
	}

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
		if err := r.resolvePackage(req.Name, result, visited); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// resolvePackage recursively resolves a package and its dependencies
func (r *Resolver) resolvePackage(name string, result *Result, visited map[string]bool) error {
	if visited[name] {
		return nil
	}
	visited[name] = true

	info, err := r.db.GetInfo(name)
	if err != nil {
		return err
	}
	if info == nil {
		return nil // Package not found
	}

	// Resolve dependencies first
	for _, dep := range info.Depends {
		if err := r.resolvePackage(dep, result, visited); err != nil {
			return err
		}
	}

	// Add this package
	result.ToInstall = append(result.ToInstall, *info)
	return nil
}