package resolver_test

import (
	"testing"

	"github.com/theshedman/shedman/pkg/core/pkgdb"
	"github.com/theshedman/shedman/pkg/core/resolver"
)

func TestDependencyTree_Build_SinglePackage(t *testing.T) {
	db := &depsTestDB{
		packages: map[string]pkgdb.PackageInfo{
			"neovim": {Name: "neovim", Version: "0.10.0", Source: pkgdb.SourceOfficial},
		},
	}

	tree := resolver.NewDependencyTree(db)
	err := tree.Build([]string{"neovim"})

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	nodes := tree.GetNodes()
	if len(nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(nodes))
	}
}

func TestDependencyTree_Build_WithDependencies(t *testing.T) {
	db := &depsTestDB{
		packages: map[string]pkgdb.PackageInfo{
			"neovim":  {Name: "neovim", Version: "0.10.0", Depends: []string{"luajit", "msgpack"}},
			"luajit":  {Name: "luajit", Version: "2.1"},
			"msgpack": {Name: "msgpack", Version: "1.0"},
		},
	}

	tree := resolver.NewDependencyTree(db)
	err := tree.Build([]string{"neovim"})

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	nodes := tree.GetNodes()
	if len(nodes) != 3 {
		t.Errorf("Expected 3 nodes (neovim + 2 deps), got %d", len(nodes))
	}
}

func TestDependencyTree_Build_NestedDependencies(t *testing.T) {
	db := &depsTestDB{
		packages: map[string]pkgdb.PackageInfo{
			"A": {Name: "A", Depends: []string{"B"}},
			"B": {Name: "B", Depends: []string{"C"}},
			"C": {Name: "C"},
		},
	}

	tree := resolver.NewDependencyTree(db)
	err := tree.Build([]string{"A"})

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	nodes := tree.GetNodes()
	if len(nodes) != 3 {
		t.Errorf("Expected 3 nodes (A → B → C), got %d", len(nodes))
	}
}

func TestDependencyTree_GetDependencies(t *testing.T) {
	db := &depsTestDB{
		packages: map[string]pkgdb.PackageInfo{
			"neovim": {Name: "neovim", Depends: []string{"luajit"}},
			"luajit": {Name: "luajit"},
		},
	}

	tree := resolver.NewDependencyTree(db)
	tree.Build([]string{"neovim"})

	deps := tree.GetDependencies("neovim")
	if len(deps) != 1 || deps[0] != "luajit" {
		t.Errorf("Expected [luajit], got %v", deps)
	}
}

// ============== Phase 3 Tests ==============

func TestDependencyTree_CircularDependency(t *testing.T) {
	// A → B → C → A (circular)
	db := &depsTestDB{
		packages: map[string]pkgdb.PackageInfo{
			"A": {Name: "A", Depends: []string{"B"}},
			"B": {Name: "B", Depends: []string{"C"}},
			"C": {Name: "C", Depends: []string{"A"}},
		},
	}

	tree := resolver.NewDependencyTree(db)
	err := tree.Build([]string{"A"})

	// Should detect circular dependency and return error
	if err == nil {
		t.Error("Expected circular dependency error")
	}
	if err != nil && err != resolver.ErrCircularDependency {
		// Check if it's wrapped
		if !resolver.IsCircularDependency(err) {
			t.Errorf("Expected ErrCircularDependency, got %v", err)
		}
	}
}

func TestDependencyTree_OptionalDependencies(t *testing.T) {
	db := &depsTestDB{
		packages: map[string]pkgdb.PackageInfo{
			"neovim": {
				Name:       "neovim",
				Depends:    []string{"luajit"},
				OptDepends: []string{"python: python support", "nodejs: node support"},
			},
			"luajit": {Name: "luajit"},
			"python": {Name: "python"},
			"nodejs": {Name: "nodejs"},
		},
	}

	tree := resolver.NewDependencyTree(db)
	tree.Build([]string{"neovim"})

	optDeps := tree.GetOptionalDependencies("neovim")
	if len(optDeps) != 2 {
		t.Errorf("Expected 2 optional deps, got %d", len(optDeps))
	}
}

func TestDependencyTree_ProvidesVirtualPackage(t *testing.T) {
	// libjson-c provides json-c
	db := &depsTestDB{
		packages: map[string]pkgdb.PackageInfo{
			"app":       {Name: "app", Depends: []string{"json-c"}},
			"libjson-c": {Name: "libjson-c", Provides: []string{"json-c"}},
		},
		provides: map[string]string{
			"json-c": "libjson-c",
		},
	}

	tree := resolver.NewDependencyTree(db)
	err := tree.Build([]string{"app"})

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Should have resolved json-c to libjson-c
	nodes := tree.GetNodes()
	if _, ok := nodes["libjson-c"]; !ok {
		t.Error("Expected libjson-c to be in tree (provides json-c)")
	}
}

func TestDependencyTree_VersionConstrainedDependency(t *testing.T) {
	db := &depsTestDB{
		packages: map[string]pkgdb.PackageInfo{
			"app": {Name: "app", Depends: []string{"lib>=2.0"}},
			"lib": {Name: "lib", Version: "2.5.0"},
		},
	}

	tree := resolver.NewDependencyTree(db)
	err := tree.Build([]string{"app"})

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	nodes := tree.GetNodes()
	if _, ok := nodes["lib"]; !ok {
		t.Error("Expected lib to be in tree")
	}
}

func TestDependencyTree_VersionConstraintNotSatisfied(t *testing.T) {
	db := &depsTestDB{
		packages: map[string]pkgdb.PackageInfo{
			"app": {Name: "app", Depends: []string{"lib>=3.0"}},
			"lib": {Name: "lib", Version: "2.5.0"},
		},
	}

	tree := resolver.NewDependencyTree(db)
	err := tree.Build([]string{"app"})

	// Should fail because lib 2.5.0 doesn't satisfy >=3.0
	if err == nil {
		t.Error("Expected version constraint error")
	}
}

func TestDependencyTree_GetInstallOrder(t *testing.T) {
	db := &depsTestDB{
		packages: map[string]pkgdb.PackageInfo{
			"A": {Name: "A", Depends: []string{"B", "C"}},
			"B": {Name: "B", Depends: []string{"D"}},
			"C": {Name: "C", Depends: []string{"D"}},
			"D": {Name: "D"},
		},
	}

	tree := resolver.NewDependencyTree(db)
	tree.Build([]string{"A"})

	order := tree.GetInstallOrder()
	// D should come before B and C, which should come before A
	dIdx, bIdx, cIdx, aIdx := -1, -1, -1, -1
	for i, pkg := range order {
		switch pkg {
		case "D":
			dIdx = i
		case "B":
			bIdx = i
		case "C":
			cIdx = i
		case "A":
			aIdx = i
		}
	}

	if dIdx > bIdx || dIdx > cIdx {
		t.Error("D should come before B and C")
	}
	if bIdx > aIdx || cIdx > aIdx {
		t.Error("B and C should come before A")
	}
}

// depsTestDB for testing (different name to avoid conflict)
type depsTestDB struct {
	packages map[string]pkgdb.PackageInfo
	provides map[string]string // virtual → real package mapping
}

func (m *depsTestDB) Search(query string) ([]pkgdb.PackageInfo, error) {
	var results []pkgdb.PackageInfo
	for _, p := range m.packages {
		results = append(results, p)
	}
	return results, nil
}

func (m *depsTestDB) GetInfo(name string) (*pkgdb.PackageInfo, error) {
	if p, ok := m.packages[name]; ok {
		return &p, nil
	}
	// Check provides mapping
	if m.provides != nil {
		if realName, ok := m.provides[name]; ok {
			if p, ok := m.packages[realName]; ok {
				return &p, nil
			}
		}
	}
	return nil, nil
}
