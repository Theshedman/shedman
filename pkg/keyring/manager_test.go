package keyring

import (
	"testing"
)

func TestNew(t *testing.T) {
	path := "/tmp/keyring"
	m := New(path)
	if m == nil {
		t.Fatal("New returned nil")
	}
	if m.path != path {
		t.Errorf("Expected path %s, got %s", path, m.path)
	}
}

func TestManager_Operations(t *testing.T) {
	m := New("/tmp/keyring")

	// Test List
	keys, err := m.List()
	if err != nil {
		t.Errorf("List returned error: %v", err)
	}
	if keys != nil {
		t.Error("Expected nil keys (stub), got something else")
	}

	// Test Add
	if err := m.Add("test-id"); err != nil {
		t.Errorf("Add returned error: %v", err)
	}

	// Test Remove
	if err := m.Remove("test-id"); err != nil {
		t.Errorf("Remove returned error: %v", err)
	}
}
