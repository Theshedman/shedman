package backends_test

import (
"testing"

"github.com/theshedman/shedman/pkg/shedman/backends"
)

func TestPacmanBackend_Name(t *testing.T) {
	backend := backends.NewPacmanBackend()
	
	if backend.Name() != "pacman" {
		t.Errorf("expected name 'pacman', got '%s'", backend.Name())
	}
}

func TestPacmanBackend_Sync(t *testing.T) {
	backend := backends.NewPacmanBackend()
	
	err := backend.Sync()
	if err != nil {
		t.Logf("Sync error (expected if pacman not available): %v", err)
	}
}
