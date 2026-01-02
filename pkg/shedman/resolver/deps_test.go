package resolver_test

import (
	"testing"

	"github.com/theshedman/shedman/pkg/shedman/pkgdb"
	"github.com/theshedman/shedman/pkg/shedman/resolver"
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

// depsTestDB for testing (different name to avoid conflict)
type depsTestDB struct {
	packages map[string]pkgdb.PackageInfo
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
	return nil, nil
}
