package theme

import (
	"strings"

	"github.com/theshedman/shedman/pkg/core"
)

// Manager handles theme operations
type Manager struct {
	core *core.Engine
}

// New creates a new theme manager
func New(c *core.Engine) *Manager {
	return &Manager{
		core: c,
	}
}

// Theme represents a system theme
type Theme struct {
	Name        string
	Description string
	Version     string
}

// Apply applies the specified theme (installs the package)
func (m *Manager) Apply(name string) error {
	pkgName := name
	if !strings.HasPrefix(name, "shedos-theme-") {
		pkgName = "shedos-theme-" + name
	}

	opts := core.InstallOptions{
		Needed:    true,
		NoConfirm: false,
	}

	return m.core.Install([]string{pkgName}, opts)
}

// List lists available themes (packages starting with shedos-theme-)
func (m *Manager) List() ([]Theme, error) {
	// Search for themes package prefix
	results, err := m.core.Search("shedos-theme-")
	if err != nil {
		return nil, err
	}

	var themes []Theme
	seen := make(map[string]bool)

	for _, p := range results {
		if strings.HasPrefix(p.Name, "shedos-theme-") {
			if seen[p.Name] {
				continue
			}
			seen[p.Name] = true
			themes = append(themes, Theme{
				Name:        p.Name,
				Description: p.Description,
				Version:     p.Version,
			})
		}
	}

	return themes, nil
}
