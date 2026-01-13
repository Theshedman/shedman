package cmd

import (
	"testing"

	"github.com/theshedman/shedman/internal/config"
)

func TestNewEngineWithConfig_NilConfig(t *testing.T) {
	// This test ensures that passing nil config does not panic
	// and returns a valid engine (using defaults)
	engine, err := NewEngineWithConfig(nil)
	if err != nil {
		t.Fatalf("NewEngineWithConfig(nil) returned error: %v", err)
	}
	if engine == nil {
		t.Fatal("NewEngineWithConfig(nil) returned nil engine")
	}
}

func TestNewEngineWithConfig_WithConfig(t *testing.T) {
	cfg := config.Default()
	engine, err := NewEngineWithConfig(cfg)
	if err != nil {
		t.Fatalf("NewEngineWithConfig(cfg) returned error: %v", err)
	}
	if engine == nil {
		t.Fatal("NewEngineWithConfig(cfg) returned nil engine")
	}
}
