package shedman

import (
"fmt"
"strings"
)

type Engine struct {
	backends []PackageBackend
}

func NewEngine() *Engine {
	return &Engine{
		backends: []PackageBackend{},
	}
}

func (e *Engine) AddBackend(backend PackageBackend) {
	e.backends = append(e.backends, backend)
}

func (e *Engine) Sync() error {
	var errors []string

	for _, backend := range e.backends {
		if err := backend.Sync(); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", backend.Name(), err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("sync failed for backends: %s", strings.Join(errors, "; "))
	}

	return nil
}
