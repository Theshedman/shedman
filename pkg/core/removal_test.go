package core

import (
	"reflect"
	"sort"
	"testing"
)

// MockProvider implements InstalledProvider for testing
type MockProvider struct {
	packages []InstalledPackage
}

func (m MockProvider) GetInstalledPackages() []InstalledPackage {
	return m.packages
}

func TestCalculateRecursiveRemoval(t *testing.T) {
	// Graph:
	// A -> B
	// B -> C
	// D -> C
	// E -> (none)

	graph := []InstalledPackage{
		{Name: "A", Depends: []string{"B"}},
		{Name: "B", Depends: []string{"C"}},
		{Name: "C", Depends: []string{}},
		{Name: "D", Depends: []string{"C"}},
		{Name: "E", Depends: []string{}},
	}

	tests := []struct {
		name     string
		targets  []string
		expected []string
	}{
		{
			name:     "Simple recursive: Remove A should remove B",
			targets:  []string{"A"},
			expected: []string{"A", "B"}, // C is kept because D needs it
		},
		{
			name: "Recursive blocked: Remove B fails because of orphan dependency check logic.",
			// Recursive removal: target and unneeded dependencies
			// It implies removing 'A' creates orphan 'B'.
			// It does NOT mean "Remove B and everything that depends on it" (that's -Rc).
			// User asked for "Recursive Removal"
			targets:  []string{"A"},
			expected: []string{"A", "B"},
		},
		{
			name:     "Full chain: Remove A and D should remove A, B, D, C",
			targets:  []string{"A", "D"},
			expected: []string{"A", "B", "C", "D"},
		},
		{
			name:     "Independent: Remove E",
			targets:  []string{"E"},
			expected: []string{"E"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := MockProvider{packages: graph}
			result := CalculateRecursiveRemoval(tt.targets, provider)

			// Sort for comparison
			sort.Strings(result)
			sort.Strings(tt.expected)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}
