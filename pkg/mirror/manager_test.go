package mirror

import (
	"errors"
	"testing"
	"time"
)

// MockBackend for Mirror testing
type MockBackend struct {
	mirrors  []Mirror
	selected bool
}

func (m *MockBackend) Name() string { return "mock" }

func (m *MockBackend) List() ([]Mirror, error) {
	return m.mirrors, nil
}

func (m *MockBackend) Test() ([]Mirror, error) {
	// Simulate testing updating speeds
	for i := range m.mirrors {
		m.mirrors[i].Speed = time.Millisecond * 100
	}
	return m.mirrors, nil
}

func (m *MockBackend) Select(topN int, countries []string, sort string) error {
	if topN <= 0 {
		return errors.New("invalid topN")
	}
	m.selected = true
	return nil
}

func (m *MockBackend) IsAvailable() bool { return true }

func TestManager_Select(t *testing.T) {
	mock := &MockBackend{
		mirrors: []Mirror{
			{URL: "http://mirror1", Country: "US"},
			{URL: "http://mirror2", Country: "DE"},
		},
	}
	mgr := NewWithBackend(mock)

	// Test Select
	err := mgr.Select(5, []string{"US"}, "rate")
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}

	if !mock.selected {
		t.Error("Select should have marked backend as selected")
	}
}

func TestManager_List(t *testing.T) {
	mock := &MockBackend{
		mirrors: []Mirror{
			{URL: "http://mirror1", Country: "US"},
		},
	}
	mgr := NewWithBackend(mock)

	list, err := mgr.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 1 {
		t.Errorf("Expected 1 mirror, got %d", len(list))
	}
}
