package config

import (
	"fmt"
	"os"
)

// DefaultApplier wraps ConfigEngine to provide high-level application logic
type DefaultApplier struct {
	engine *ConfigEngine
}

// NewDefaultApplier creates a new applier
func NewDefaultApplier(engine *ConfigEngine) *DefaultApplier {
	return &DefaultApplier{engine: engine}
}

// Apply resolves owner and applies package defaults to the target path
func (a *DefaultApplier) Apply(path string) error {
	owner, err := a.engine.GetFileOwner(path)
	if err != nil {
		return fmt.Errorf("failed to identify owner object: %w", err)
	}

	original, err := a.engine.GetOriginal(path)
	if err != nil {
		return fmt.Errorf("failed to get original content: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "shedman-apply-*.conf")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
	}()

	if _, err := tmpFile.Write(original); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	return a.engine.Apply(owner, tmpFile.Name(), path)
}
