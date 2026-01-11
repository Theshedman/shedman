package notifier

import (
	"testing"
)

// MockBackend restricts dependencies for testing
type MockBackend struct {
	lastTitle string
	lastMsg   string
	lastLevel string
}

func (m *MockBackend) Name() string { return "mock" }

func (m *MockBackend) Notify(title, message, level string) error {
	m.lastTitle = title
	m.lastMsg = message
	m.lastLevel = level
	return nil
}

func (m *MockBackend) IsAvailable() bool { return true }

func TestManager_Notify(t *testing.T) {
	mock := &MockBackend{}
	mgr := NewWithBackend(mock)

	err := mgr.Notify("Update Available", "5 packages to update", "info")
	if err != nil {
		t.Fatalf("Notify failed: %v", err)
	}

	if mock.lastTitle != "Update Available" {
		t.Errorf("Expected title 'Update Available', got '%s'", mock.lastTitle)
	}
	if mock.lastMsg != "5 packages to update" {
		t.Errorf("Expected message '5 packages to update', got '%s'", mock.lastMsg)
	}
	if mock.lastLevel != "info" {
		t.Errorf("Expected level 'info', got '%s'", mock.lastLevel)
	}
}
