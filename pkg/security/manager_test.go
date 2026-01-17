package security

import (
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

func TestNew(t *testing.T) {
	// core can be nil for this stub test as it's not used yet
	var engine *core.Engine
	s := New(engine)
	if s == nil {
		t.Fatal("New returned nil")
	}
	if s.core != engine {
		t.Error("Engine not set correctly")
	}
}

func TestScanner_Check(t *testing.T) {
	s := New(nil)

	vulns, err := s.Check()
	if err != nil {
		t.Errorf("Check returned error: %v", err)
	}
	if vulns != nil {
		t.Error("Expected nil vulnerabilies (stub), got something else")
	}
}
