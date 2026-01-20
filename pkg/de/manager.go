package de

import (
	"fmt"
	"strings"

	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/snapshot"
	"github.com/theshedman/shedman/pkg/system"
)

// ConfigApplier handles configuration application
type ConfigApplier interface {
	Apply(path string) error
}

// Manager handles desktop environment operations
type Manager struct {
	core    *core.Engine
	snapMgr snapshot.Manager
	svcMgr  system.ServiceManager
	applier ConfigApplier
}

// SwitchOptions configuration for DE switch
type SwitchOptions struct {
	NoSnapshot bool
	KeepOld    bool
	NoConfirm  bool
	DryRun     bool
}

// New creates a new DE manager
func New(c *core.Engine) *Manager {
	return &Manager{
		core: c,
	}
}

// SetSnapshotManager sets the snapshot manager dependency
func (m *Manager) SetSnapshotManager(s snapshot.Manager) {
	m.snapMgr = s
}

// SetServiceManager sets the service manager dependency
func (m *Manager) SetServiceManager(s system.ServiceManager) {
	m.svcMgr = s
}

// SetConfigApplier sets the config applier dependency
func (m *Manager) SetConfigApplier(a ConfigApplier) {
	m.applier = a
}

// DesktopEnvironment represents a DE
type DesktopEnvironment struct {
	ID        string
	Name      string
	Group     string
	Package   string // Derived from group
	Service   string // Display Manager Service
	Installed bool
}

// deMetadata holds static config not in groups
type deMetadata struct {
	Name    string
	Service string
}

// Known DEs (Metadata only - packages come from core.DefaultGroups)
var deMetadataRegistry = map[string]deMetadata{
	"hyprland": {Name: "Hyprland", Service: "sddm"},
	"gnome":    {Name: "GNOME", Service: "gdm"},
	"kde":      {Name: "KDE Plasma", Service: "sddm"},
	"cosmic":   {Name: "COSMIC", Service: "gdm"},
	"budgie":   {Name: "Budgie", Service: "lightdm"},
	"cinnamon": {Name: "Cinnamon", Service: "lightdm"},
	"deepin":   {Name: "Deepin", Service: "lightdm"},
	"mate":     {Name: "MATE", Service: "lightdm"},
	"pantheon": {Name: "Pantheon", Service: "lightdm"},
}

// Switch switches to the specified DE
func (m *Manager) Switch(name string, opts SwitchOptions) error {
	meta, ok := deMetadataRegistry[name]
	if !ok {
		return fmt.Errorf("unknown desktop environment: %s", name)
	}

	groupName := "shedos-" + name
	group, exists := core.DefaultGroups[groupName]
	if !exists {
		return fmt.Errorf("package group %s not found in core definitions", groupName)
	}

	targetGroup := core.GroupPrefix + groupName

	if !opts.NoSnapshot && m.snapMgr != nil && !opts.DryRun {
		desc := fmt.Sprintf("Pre-switch snapshot: %s", name)
		_, err := m.snapMgr.Create(desc, snapshot.CreateOptions{
			Type: "pre",
		})
		if err != nil {
			return fmt.Errorf("failed to create snapshot: %w", err)
		}
	}

	var currentDEGroups []string
	if !opts.KeepOld {
		known, _ := m.List()
		for _, de := range known {
			if de.Installed && de.ID != name {
				currentDEGroups = append(currentDEGroups, de.Group)
			}
		}
	}

	if !opts.DryRun {
		err := m.core.Install([]string{targetGroup}, core.InstallOptions{
			Needed:    true,
			NoConfirm: opts.NoConfirm,
		})
		if err != nil {
			return fmt.Errorf("failed to install %s: %w", targetGroup, err)
		}
	}

	if !opts.KeepOld && len(currentDEGroups) > 0 && !opts.DryRun {
		var pkgsToRemove []string
		for _, gName := range currentDEGroups {
			rawName := gName[1:]
			if g, ok := core.DefaultGroups[rawName]; ok {
				pkgsToRemove = append(pkgsToRemove, g.Packages...)
			}
		}

		if len(pkgsToRemove) > 0 {
			_ = m.core.Remove(pkgsToRemove, core.RemoveOptions{
				Recursive: true,
				NoConfirm: opts.NoConfirm,
			})
		}
	}

	if m.applier != nil && !opts.DryRun {
		for _, configPath := range group.Configs {
			if err := m.applier.Apply(configPath); err != nil {
				return fmt.Errorf("failed to apply config %s: %w", configPath, err)
			}
		}
	}

	if m.svcMgr != nil && meta.Service != "" && !opts.DryRun {
		if !opts.KeepOld {
			for _, gName := range currentDEGroups {
				id := strings.TrimPrefix(gName, "@shedos-")
				if oldMeta, ok := deMetadataRegistry[id]; ok && oldMeta.Service != "" {
					if oldMeta.Service != meta.Service {
						_ = m.svcMgr.Disable(oldMeta.Service)
					}
				}
			}
		}

		if err := m.svcMgr.Enable(meta.Service); err != nil {
			return fmt.Errorf("failed to enable display manager %s: %w", meta.Service, err)
		}
	}

	return nil
}

// List lists available DEs
func (m *Manager) List() ([]DesktopEnvironment, error) {
	var des []DesktopEnvironment

	for id, meta := range deMetadataRegistry {
		groupName := "shedos-" + id
		group, exists := core.DefaultGroups[groupName]
		if !exists {
			// Skip if group definition missing
			continue
		}

		// Determine "Main Package" for installation check
		// Heuristic: First package in the list
		mainPkg := ""
		if len(group.Packages) > 0 {
			mainPkg = group.Packages[0]
		}

		installed := false
		if mainPkg != "" {
			installed = m.core.IsInstalled(mainPkg)
		}

		des = append(des, DesktopEnvironment{
			ID:        id,
			Name:      meta.Name,
			Group:     core.GroupPrefix + groupName,
			Package:   mainPkg,
			Service:   meta.Service,
			Installed: installed,
		})
	}

	return des, nil
}
