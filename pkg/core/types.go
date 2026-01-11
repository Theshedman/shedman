// Package backend provides registry and factory functions for package manager backends.
package core

import (
	"fmt"
	"sync"

	"github.com/theshedman/shedman/internal/config"
)

// BackendFactory is a function that creates a backend with configuration
type BackendFactory func(cfg *config.BackendConfig) (OfficialBackend, error)

// registry holds registered backend factories
var (
	registryMu sync.RWMutex
	factories  = make(map[string]BackendFactory)
)

// RegisterBackend registers a backend factory by name
func RegisterBackend(name string, factory BackendFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	factories[name] = factory
}

// GetRegisteredBackend returns a backend factory by name
func GetRegisteredBackend(name string) (BackendFactory, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	factory, ok := factories[name]
	return factory, ok
}

// ListRegisteredBackends returns all registered backend names
func ListRegisteredBackends() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(factories))
	for name := range factories {
		names = append(names, name)
	}
	return names
}

// CreateBackend creates a backend by name with configuration
func CreateBackend(name string, cfg *config.BackendConfig) (OfficialBackend, error) {
	factory, ok := GetRegisteredBackend(name)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrBackendNotFound, name)
	}
	return factory(cfg)
}

// DetectBackendWithConfig auto-detects and returns the appropriate backend
// using configuration to override detection if specified
func DetectBackendWithConfig(cfg *config.BackendConfig) (OfficialBackend, error) {
	// If override is set, use that backend directly
	if cfg != nil && cfg.Override != "" {
		factory, ok := GetRegisteredBackend(cfg.Override)
		if !ok {
			return nil, fmt.Errorf("%w: override backend '%s' not registered", ErrBackendNotFound, cfg.Override)
		}
		return factory(cfg)
	}

	// Auto-detect based on distribution
	if cfg == nil || cfg.AutoDetect {
		return autoDetectBackend(cfg)
	}

	return nil, ErrBackendNotFound
}

// autoDetectBackend detects the backend based on system information
func autoDetectBackend(cfg *config.BackendConfig) (OfficialBackend, error) {
	info := DetectDistro()

	// Map family to backend name
	var backendName string
	if info.Family == "arch" {
		backendName = "pacman"
	} else {
		// Fallback to binary detection
		return detectByBinaryWithConfig(cfg)
	}

	// Check if the backend is registered
	factory, ok := GetRegisteredBackend(backendName)
	if !ok {
		return nil, fmt.Errorf("%w: detected backend '%s' not registered", ErrBackendNotFound, backendName)
	}

	return factory(cfg)
}

// detectByBinaryWithConfig attempts to detect backend by available binaries
func detectByBinaryWithConfig(cfg *config.BackendConfig) (OfficialBackend, error) {
	// Check binaries in priority order
	candidates := []string{"pacman"}

	for _, name := range candidates {
		if hasBinary(name) {
			if factory, ok := GetRegisteredBackend(name); ok {
				return factory(cfg)
			}
		}
	}

	return nil, ErrBackendNotFound
}
