package shedman_test

import (
"testing"

"github.com/theshedman/shedman/pkg/shedman"
)

// MockBackend is a test implementation of PackageBackend
type MockBackend struct {
	SyncCalled bool
}

func (m *MockBackend) Name() string {
	return "mock"
}

func (m *MockBackend) Sync() error {
	m.SyncCalled = true
	return nil
}

func TestEngine_Sync(t *testing.T) {
	// Arrange
	mock := &MockBackend{}
	engine := shedman.NewEngine()
	engine.AddBackend(mock)

	// Act
	err := engine.Sync()

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !mock.SyncCalled {
		t.Error("expected backend Sync() to be called, but it wasn't")
	}
}

func TestBackend_Name(t *testing.T) {
	mock := &MockBackend{}
	
	if mock.Name() != "mock" {
		t.Errorf("expected name 'mock', got '%s'", mock.Name())
	}
}
