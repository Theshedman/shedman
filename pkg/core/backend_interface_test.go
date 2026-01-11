package core_test

import (
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

// TestBackendInterface_CompileTime ensures the Backend interface exists
// and has the expected method signatures.
// This is a compile-time check primarily, but wrapped in a test.
func TestBackendInterface_CompileTime(t *testing.T) {
	var _ core.Backend = (*mockBackend)(nil)
}

// mockBackend implements core.Backend for testing
type mockBackend struct{}

func (m *mockBackend) Name() string      { return "mock" }
func (m *mockBackend) Sync() error       { return nil }
func (m *mockBackend) IsAvailable() bool { return true }
