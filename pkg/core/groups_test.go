package core

import (
	"testing"

)

func TestIsGroupReference(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"@base", true},
		{"@dev", true},
		{"firefox", false},
		{"base", false},
		{"@", true},
	}

	for _, tc := range tests {
		result := IsGroupReference(tc.input)
		if result != tc.expected {
			t.Errorf("IsGroupReference(%q) = %v, want %v", tc.input, result, tc.expected)
		}
	}
}

func TestGroupRegistry_GetGroup(t *testing.T) {
	registry := NewGroupRegistry()

	// Test with prefix
	group, exists := registry.GetGroup("@base")
	if !exists {
		t.Fatal("Expected @base group to exist")
	}
	if group.Name != "base" {
		t.Errorf("Expected name 'base', got %s", group.Name)
	}

	// Test without prefix
	group, exists = registry.GetGroup("dev")
	if !exists {
		t.Fatal("Expected dev group to exist")
	}
	if group.Name != "dev" {
		t.Errorf("Expected name 'dev', got %s", group.Name)
	}

	// Test non-existent group
	_, exists = registry.GetGroup("@nonexistent")
	if exists {
		t.Error("Expected nonexistent group to not exist")
	}
}

func TestGroupRegistry_ExpandGroups(t *testing.T) {
	registry := NewGroupRegistry()

	// Test mixed packages and groups
	packages, err := registry.ExpandGroups([]string{"firefox", "@fonts"})
	if err != nil {
		t.Fatalf("ExpandGroups failed: %v", err)
	}

	if len(packages) < 2 {
		t.Errorf("Expected at least 2 packages, got %d", len(packages))
	}

	// First package should be firefox
	if packages[0] != "firefox" {
		t.Errorf("Expected first package 'firefox', got %s", packages[0])
	}
}

func TestGroupRegistry_ExpandGroups_Unknown(t *testing.T) {
	registry := NewGroupRegistry()

	_, err := registry.ExpandGroups([]string{"@unknown-group"})
	if err == nil {
		t.Error("Expected error for unknown group")
	}
}

func TestGroupRegistry_ListGroups(t *testing.T) {
	registry := NewGroupRegistry()
	groups := registry.ListGroups()

	if len(groups) < 10 {
		t.Errorf("Expected at least 10 groups, got %d", len(groups))
	}
}

func TestGroupRegistry_ExpandGroupsWithOptional(t *testing.T) {
	registry := NewGroupRegistry()

	required, optional, err := registry.ExpandGroupsWithOptional([]string{"@jvm-dev"}, true)
	if err != nil {
		t.Fatalf("ExpandGroupsWithOptional failed: %v", err)
	}

	if len(required) < 1 {
		t.Error("Expected at least 1 required package")
	}

	// jvm-dev has optional packages (kotlin, scala)
	if len(optional) < 1 {
		t.Error("Expected at least 1 optional package for @jvm-dev")
	}
}

func TestGroupRegistry_GetGroupDescription(t *testing.T) {
	registry := NewGroupRegistry()
	desc := registry.GetGroupDescription("base")

	if desc == "" {
		t.Error("Expected non-empty description")
	}
	if !contains(desc, "base") {
		t.Error("Description should contain group name")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
