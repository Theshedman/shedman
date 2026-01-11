package core

import (
	"fmt"
	"strings"
	"testing"
)

// MockBackend implements PackageBackend for testing
type MockBackend struct {
	name       string
	shouldFail bool
}

func (m *MockBackend) Sync() error {
	if m.shouldFail {
		return fmt.Errorf("%s sync failed", m.name)
	}
	return nil
}

func (m *MockBackend) Name() string {
	return m.name
}

func TestEngine_Sync(t *testing.T) {
	engine := NewEngine()
	backend := &MockBackend{name: "test", shouldFail: false}
	engine.AddBackend(backend)

	err := engine.Sync()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestBackend_Name(t *testing.T) {
	backend := &MockBackend{name: "test-backend", shouldFail: false}

	if backend.Name() != "test-backend" {
		t.Errorf("expected name 'test-backend', got '%s'", backend.Name())
	}
}

func TestEngine_Sync_CollectsAllErrors(t *testing.T) {
	engine := NewEngine()

	// Add two backends that both fail
	engine.AddBackend(&MockBackend{name: "backend1", shouldFail: true})
	engine.AddBackend(&MockBackend{name: "backend2", shouldFail: true})

	err := engine.Sync()

	if err == nil {
		t.Error("Sync should return error when backends fail")
	}

	// Error should mention both backends
	errStr := err.Error()
	if !strings.Contains(errStr, "backend1") {
		t.Error("error should mention backend1")
	}
	if !strings.Contains(errStr, "backend2") {
		t.Error("error should mention backend2")
	}
}

func TestEngine_Sync_ContinuesAfterError(t *testing.T) {
	engine := NewEngine()

	// First backend fails, second succeeds
	backend1 := &MockBackend{name: "failing", shouldFail: true}
	backend2 := &MockBackend{name: "success", shouldFail: false}

	engine.AddBackend(backend1)
	engine.AddBackend(backend2)

	err := engine.Sync()

	// Should still get an error (from backend1)
	if err == nil {
		t.Error("Sync should return error when a backend fails")
	}

	// But the error should only mention the failing backend
	errStr := err.Error()
	if !strings.Contains(errStr, "failing") {
		t.Error("error should mention failing backend")
	}
}

func TestEngine_GetOfficialBackend_Nil(t *testing.T) {
	engine := NewEngine()
	if engine.GetOfficialBackend() != nil {
		t.Error("Expected nil official backend for new engine")
	}
}

func TestEngine_NewEngineWithBackend(t *testing.T) {
	mock := &MockBackend{name: "mock", shouldFail: false}
	// MockBackend doesn't implement OfficialBackend, so we test with nil
	engine := NewEngineWithBackend(nil)

	if engine.GetOfficialBackend() != nil {
		t.Error("Expected nil backend when created with nil")
	}

	// Add regular backend
	engine.AddBackend(mock)
	if err := engine.Sync(); err != nil {
		t.Errorf("Sync failed: %v", err)
	}
}

func TestEngine_IsOfficialBackendAvailable_NoBackend(t *testing.T) {
	engine := NewEngine()
	if engine.IsOfficialBackendAvailable() {
		t.Error("Expected false when no backend is set")
	}
}
