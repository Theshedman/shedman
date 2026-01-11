package resolver_test

import (
	"testing"

	"github.com/theshedman/shedman/pkg/shedman/resolver"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		v1, v2   string
		expected int
	}{
		// Basic comparisons
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.0.1", "1.0.0", 1},
		{"2.0.0", "1.9.9", 1},
		{"1.10.0", "1.9.0", 1},
		{"1.2.3", "1.2.3.4", -1},
		{"0.10.0", "0.9.5", 1},
		{"v1.0.0", "1.0.0", 0},

		// Epoch comparisons
		{"1:1.0", "1:0.9", 1},
		{"2:1.0", "1:9.0", 1},

		// Package release
		{"1.0-1", "1.0-2", -1},
		{"1.0.0-3", "1.0.0-1", 1},

		// Pre-release comparisons
		{"1.0.0-alpha", "1.0.0", -1},         // pre-release < release
		{"1.0.0", "1.0.0-alpha", 1},          // release > pre-release
		{"1.0.0-alpha", "1.0.0-beta", -1},    // alpha < beta
		{"1.0.0-beta", "1.0.0-rc", -1},       // beta < rc
		{"1.0.0-rc", "1.0.0-rc2", -1},        // rc < rc2
		{"1.0.0-rc1", "1.0.0-rc2", -1},       // rc1 < rc2
		{"1.0.0-alpha", "1.0.0-alpha", 0},    // same pre-release
		{"1.0.0-alpha1", "1.0.0-alpha2", -1}, // alpha1 < alpha2
	}

	for _, tt := range tests {
		t.Run(tt.v1+" vs "+tt.v2, func(t *testing.T) {
			result := resolver.CompareVersions(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}

func TestMatchesVersionConstraint_Equal(t *testing.T) {
	if !resolver.MatchesVersionConstraint("1.0.0", "1.0.0", resolver.OpEqual) {
		t.Error("Expected 1.0.0 = 1.0.0 to match")
	}
	if resolver.MatchesVersionConstraint("1.0.0", "1.0.1", resolver.OpEqual) {
		t.Error("Expected 1.0.0 = 1.0.1 to not match")
	}
}

func TestMatchesVersionConstraint_GreaterEqual(t *testing.T) {
	if !resolver.MatchesVersionConstraint("1.0.0", "1.0.0", resolver.OpGreaterEqual) {
		t.Error("Expected 1.0.0 >= 1.0.0 to match")
	}
	if !resolver.MatchesVersionConstraint("1.0.1", "1.0.0", resolver.OpGreaterEqual) {
		t.Error("Expected 1.0.1 >= 1.0.0 to match")
	}
	if resolver.MatchesVersionConstraint("0.9.9", "1.0.0", resolver.OpGreaterEqual) {
		t.Error("Expected 0.9.9 >= 1.0.0 to not match")
	}
}

func TestMatchesVersionConstraint_LessEqual(t *testing.T) {
	if !resolver.MatchesVersionConstraint("1.0.0", "1.0.0", resolver.OpLessEqual) {
		t.Error("Expected 1.0.0 <= 1.0.0 to match")
	}
	if !resolver.MatchesVersionConstraint("0.9.0", "1.0.0", resolver.OpLessEqual) {
		t.Error("Expected 0.9.0 <= 1.0.0 to match")
	}
	if resolver.MatchesVersionConstraint("1.0.1", "1.0.0", resolver.OpLessEqual) {
		t.Error("Expected 1.0.1 <= 1.0.0 to not match")
	}
}

func TestMatchesVersionConstraint_Greater(t *testing.T) {
	if resolver.MatchesVersionConstraint("1.0.0", "1.0.0", resolver.OpGreater) {
		t.Error("Expected 1.0.0 > 1.0.0 to not match")
	}
	if !resolver.MatchesVersionConstraint("1.0.1", "1.0.0", resolver.OpGreater) {
		t.Error("Expected 1.0.1 > 1.0.0 to match")
	}
}

func TestMatchesVersionConstraint_Less(t *testing.T) {
	if resolver.MatchesVersionConstraint("1.0.0", "1.0.0", resolver.OpLess) {
		t.Error("Expected 1.0.0 < 1.0.0 to not match")
	}
	if !resolver.MatchesVersionConstraint("0.9.0", "1.0.0", resolver.OpLess) {
		t.Error("Expected 0.9.0 < 1.0.0 to match")
	}
}

func TestMatchesVersionConstraint_NoConstraint(t *testing.T) {
	// Empty constraint should match any version
	if !resolver.MatchesVersionConstraint("1.0.0", "", "") {
		t.Error("Expected empty constraint to match any version")
	}
}

func TestRequestMatchesPackage(t *testing.T) {
	req := resolver.ParseRequest("neovim>=0.9.0")

	if !resolver.RequestMatchesPackage(req, "0.10.0") {
		t.Error("Expected 0.10.0 to satisfy >=0.9.0")
	}
	if !resolver.RequestMatchesPackage(req, "0.9.0") {
		t.Error("Expected 0.9.0 to satisfy >=0.9.0")
	}
	if resolver.RequestMatchesPackage(req, "0.8.0") {
		t.Error("Expected 0.8.0 to not satisfy >=0.9.0")
	}
}
