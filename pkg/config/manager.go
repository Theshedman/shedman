package config

import (
	"strings"

	"github.com/theshedman/shedman/pkg/core"
)

// Manager handles configuration package operations
type Manager struct {
	core *core.Engine
}

// New creates a new config manager
func New(c *core.Engine) *Manager {
	return &Manager{
		core: c,
	}
}

// ConfigPackage represents a configuration package
type ConfigPackage struct {
	Name        string
	Description string
	Version     string
}

// Apply applies a configuration package (installs it)
// name can be short (e.g. "hypr") or full ("shedos-configs-hypr")
func (m *Manager) Apply(name string) error {
	pkgName := name
	if !strings.HasPrefix(name, "shedos-configs-") {
		pkgName = "shedos-configs-" + name
	}

	// Use generic install options
	opts := core.InstallOptions{
		Needed:    true,
		NoConfirm: false,
	}

	return m.core.Install([]string{pkgName}, opts)
}

// List returns available configuration packages matching "shedos-configs-*"
func (m *Manager) List() ([]ConfigPackage, error) {
	// Search for all packages starting with shedos-configs-
	results, err := m.core.Search("shedos-configs-")
	if err != nil {
		return nil, err
	}

	var configs []ConfigPackage
	seen := make(map[string]bool)

	for _, p := range results {
		if strings.HasPrefix(p.Name, "shedos-configs-") {
			if seen[p.Name] {
				continue
			}
			seen[p.Name] = true
			configs = append(configs, ConfigPackage{
				Name:        p.Name,
				Description: p.Description,
				Version:     p.Version,
			})
		}
	}

	return configs, nil
}
