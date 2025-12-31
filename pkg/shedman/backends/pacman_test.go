package backends_test

import (
"errors"
"testing"

"github.com/theshedman/shedman/pkg/shedman/backends"
)

// MockCommandExecutor simulates command execution for testing
type MockCommandExecutor struct {
	ShouldFail   bool
	LastCommand  string
	LastArgs     []string
	ExecuteCalls int
}

func (m *MockCommandExecutor) Run(name string, args ...string) error {
	m.ExecuteCalls++
	m.LastCommand = name
	m.LastArgs = args
	if m.ShouldFail {
		return errors.New("mock command failed")
	}
	return nil
}

func TestPacmanBackend_Name(t *testing.T) {
	backend := backends.NewPacmanBackend()

	if backend.Name() != "pacman" {
		t.Errorf("expected name 'pacman', got '%s'", backend.Name())
	}
}

func TestPacmanBackend_Sync_Success(t *testing.T) {
	mock := &MockCommandExecutor{ShouldFail: false}
	backend := backends.NewPacmanBackendWithExecutor("/usr/bin/pacman", mock)

	err := backend.Sync()

	if err != nil {
		t.Errorf("Sync should succeed, but got error: %v", err)
	}
	if mock.ExecuteCalls != 1 {
		t.Errorf("expected 1 execute call, got %d", mock.ExecuteCalls)
	}
	if mock.LastCommand != "/usr/bin/pacman" {
		t.Errorf("expected command '/usr/bin/pacman', got '%s'", mock.LastCommand)
	}
	if len(mock.LastArgs) != 1 || mock.LastArgs[0] != "-Sy" {
		t.Errorf("expected args ['-Sy'], got %v", mock.LastArgs)
	}
}

func TestPacmanBackend_Sync_Failure(t *testing.T) {
	mock := &MockCommandExecutor{ShouldFail: true}
	backend := backends.NewPacmanBackendWithExecutor("/usr/bin/pacman", mock)

	err := backend.Sync()

	if err == nil {
		t.Error("Sync should fail, but got nil error")
	}
}

func TestPacmanBackend_CustomBinaryPath(t *testing.T) {
	mock := &MockCommandExecutor{ShouldFail: false}
	backend := backends.NewPacmanBackendWithExecutor("/custom/path/pacman", mock)

	_ = backend.Sync()

	if mock.LastCommand != "/custom/path/pacman" {
		t.Errorf("expected custom path '/custom/path/pacman', got '%s'", mock.LastCommand)
	}
}
