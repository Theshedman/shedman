package de

import (
	"fmt"

	"github.com/theshedman/shedman/pkg/core"
)

// Manager handles desktop environment operations
type Manager struct {
	core *core.Engine
}

// New creates a new DE manager
func New(c *core.Engine) *Manager {
	return &Manager{
		core: c,
	}
}

// DesktopEnvironment represents a DE
type DesktopEnvironment struct {
	Name      string
	Installed bool
}

var dePackages = map[string]string{
	"hyprland": "hyprland",
	"gnome":    "gnome-shell",
	"kde":      "plasma-desktop",
	"cosmic":   "cosmic-session",
}

// Switch switches to the specified DE
func (m *Manager) Switch(name string) error {
	pkgName, ok := dePackages[name]
	if !ok {
		return fmt.Errorf("unknown desktop environment: %s", name)
	}

	// Install the DE package
	opts := core.InstallOptions{
		Needed:    true,
		NoConfirm: false,
	}
	return m.core.Install([]string{pkgName}, opts)
}

// List lists available DEs
func (m *Manager) List() ([]DesktopEnvironment, error) {
	var des []DesktopEnvironment

	for name, pkg := range dePackages {
		installed := m.core.IsInstalled(pkg)
		des = append(des, DesktopEnvironment{
			Name:      name,
			Installed: installed,
		})
	}

	return des, nil
}
