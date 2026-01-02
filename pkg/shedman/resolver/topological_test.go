package resolver_test

import (
	"testing"

	"github.com/theshedman/shedman/pkg/shedman/resolver"
)

func TestTopologicalSort_SinglePackage(t *testing.T) {
	nodes := map[string]*resolver.DependencyNode{
		"neovim": {Children: []string{}},
	}

	sorted := resolver.TopologicalSort(nodes)

	if len(sorted) != 1 || sorted[0] != "neovim" {
		t.Errorf("Expected [neovim], got %v", sorted)
	}
}

func TestTopologicalSort_LinearDependency(t *testing.T) {
	// A depends on B, B depends on C
	// Install order should be: C, B, A
	nodes := map[string]*resolver.DependencyNode{
		"A": {Children: []string{"B"}},
		"B": {Children: []string{"C"}},
		"C": {Children: []string{}},
	}

	sorted := resolver.TopologicalSort(nodes)

	if len(sorted) != 3 {
		t.Fatalf("Expected 3 packages, got %d", len(sorted))
	}

	// C must come before B, B must come before A
	posC := indexOf(sorted, "C")
	posB := indexOf(sorted, "B")
	posA := indexOf(sorted, "A")

	if posC > posB || posB > posA {
		t.Errorf("Wrong order: expected C before B before A, got %v", sorted)
	}
}

func TestTopologicalSort_MultipleDependencies(t *testing.T) {
	// A depends on B and C
	nodes := map[string]*resolver.DependencyNode{
		"A": {Children: []string{"B", "C"}},
		"B": {Children: []string{}},
		"C": {Children: []string{}},
	}

	sorted := resolver.TopologicalSort(nodes)

	if len(sorted) != 3 {
		t.Fatalf("Expected 3 packages, got %d", len(sorted))
	}

	// B and C must come before A
	posA := indexOf(sorted, "A")
	posB := indexOf(sorted, "B")
	posC := indexOf(sorted, "C")

	if posB > posA || posC > posA {
		t.Errorf("A should be installed after B and C, got %v", sorted)
	}
}

func TestTopologicalSort_DiamondDependency(t *testing.T) {
	// A depends on B and C, both B and C depend on D
	nodes := map[string]*resolver.DependencyNode{
		"A": {Children: []string{"B", "C"}},
		"B": {Children: []string{"D"}},
		"C": {Children: []string{"D"}},
		"D": {Children: []string{}},
	}

	sorted := resolver.TopologicalSort(nodes)

	if len(sorted) != 4 {
		t.Fatalf("Expected 4 packages, got %d", len(sorted))
	}

	// D must come first, A must come last
	posD := indexOf(sorted, "D")
	posA := indexOf(sorted, "A")

	if posD > 0 {
		t.Errorf("D should be first, got %v", sorted)
	}
	if posA != 3 {
		t.Errorf("A should be last, got %v", sorted)
	}
}

// Helper function
func indexOf(slice []string, item string) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}
	return -1
}
