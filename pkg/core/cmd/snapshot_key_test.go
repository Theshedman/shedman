package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/snapshot"
)

// Mock KeyManager
type MockKeyManager struct {
	GenerateFunc func(desc string) (string, error)
	ExportFunc   func(id string, path string) error
	ImportFunc   func(path string) error
	ListFunc     func() ([]snapshot.Key, error)
	DeleteFunc   func(id string) error
}

func (m *MockKeyManager) Generate(desc string) (string, error) {
	if m.GenerateFunc != nil {
		return m.GenerateFunc(desc)
	}
	return "key-123", nil
}
func (m *MockKeyManager) Export(id string, path string) error {
	if m.ExportFunc != nil {
		return m.ExportFunc(id, path)
	}
	return nil
}
func (m *MockKeyManager) Import(path string) error {
	if m.ImportFunc != nil {
		return m.ImportFunc(path)
	}
	return nil
}

func (m *MockKeyManager) List() ([]snapshot.Key, error) {
	if m.ListFunc != nil {
		return m.ListFunc()
	}
	return nil, nil
}

func (m *MockKeyManager) Delete(id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

func TestSnapshotKeyCmd(t *testing.T) {
	// Setup
	engine := core.NewEngine()
	mock := &MockKeyManager{
		GenerateFunc: func(desc string) (string, error) {
			if desc != "Test Key" {
				return "", nil
			}
			return "new-key-id", nil
		},
	}
	engine.SetKeyManager(mock)

	buf := new(bytes.Buffer)

	// Test Generate directly via logic function
	args := []string{"Test Key"}
	if err := RunSnapshotKeyGenerate(engine, args, buf); err != nil {
		t.Fatalf("Generate execution failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "new-key-id") {
		t.Errorf("Expected output to contain key ID, got: %s", output)
	}
}
