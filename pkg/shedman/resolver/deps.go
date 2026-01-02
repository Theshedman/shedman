package resolver

import (
	"github.com/theshedman/shedman/pkg/shedman/pkgdb"
)

// DependencyNode represents a package in the dependency tree
type DependencyNode struct {
	Package  pkgdb.PackageInfo
	Children []string
}

// DependencyTree builds and manages package dependency trees
type DependencyTree struct {
	db      pkgdb.PackageDB
	nodes   map[string]*DependencyNode
	visited map[string]bool
}

// NewDependencyTree creates a new DependencyTree
func NewDependencyTree(db pkgdb.PackageDB) *DependencyTree {
	return &DependencyTree{
		db:      db,
		nodes:   make(map[string]*DependencyNode),
		visited: make(map[string]bool),
	}
}

// Build builds the dependency tree for the given packages
func (dt *DependencyTree) Build(packages []string) error {
	for _, pkg := range packages {
		if err := dt.buildNode(pkg); err != nil {
			return err
		}
	}
	return nil
}

// buildNode recursively builds a node and its dependencies
func (dt *DependencyTree) buildNode(name string) error {
	if dt.visited[name] {
		return nil
	}
	dt.visited[name] = true

	info, err := dt.db.GetInfo(name)
	if err != nil {
		return err
	}
	if info == nil {
		return nil
	}

	node := &DependencyNode{
		Package:  *info,
		Children: info.Depends,
	}
	dt.nodes[name] = node

	// Recursively build dependencies
	for _, dep := range info.Depends {
		if err := dt.buildNode(dep); err != nil {
			return err
		}
	}

	return nil
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