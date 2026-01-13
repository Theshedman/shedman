package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/theshedman/shedman/internal/config"
)

// GroupPrefix is the prefix for package group references
const GroupPrefix = "@"

// PackageGroup represents a named collection of packages
type PackageGroup struct {
	Name        string   // Group name (without @ prefix)
	Description string   // Human-readable description
	Packages    []string // Package names in this group
	Optional    []string // Optional packages in this group
	Includes    []string // Other groups to include (nested groups)
}

// DefaultGroups contains the predefined package groups
var DefaultGroups = map[string]PackageGroup{
	// Base system
	"base": {
		Name:        "base",
		Description: "Essential base system packages",
		Packages: []string{
			"base", "base-devel", "linux", "linux-firmware",
			"networkmanager", "sudo", "vim", "git",
		},
	},

	// Development groups
	"dev": {
		Name:        "dev",
		Description: "General development tools",
		Packages: []string{
			"base-devel", "git", "vim", "neovim", "tmux",
			"curl", "wget", "jq", "ripgrep", "fd", "fzf",
			"make", "cmake", "gcc", "clang", "gdb",
			"lazygit", "grep", "sed", "gawk",
			"lua-language-server", "bash-language-server", "python-lsp-server",
			"shellcheck", "shfmt", "docker", "docker-compose",
		},
	},
	"web-dev": {
		Name:        "web-dev",
		Description: "Web development tools",
		Packages: []string{
			"nodejs", "npm", "yarn", "docker", "nginx",
		},
	},
	"python-dev": {
		Name:        "python-dev",
		Description: "Python development tools",
		Packages: []string{
			"python", "python-pip", "python-poetry", "pyenv", "ipython", "python-virtualenv",
			"python-pytest",
		},
	},
	"rust-dev": {
		Name:        "rust-dev",
		Description: "Rust development tools",
		Packages: []string{
			"rustup", "rust-analyzer",
		},
	},
	"go-dev": {
		Name:        "go-dev",
		Description: "Go development tools",
		Packages: []string{
			"go", "gopls", "delve",
		},
	},
	"jvm-dev": {
		Name:        "jvm-dev",
		Description: "JVM development tools (Java, Kotlin, Scala)",
		Packages: []string{
			"jdk-openjdk", "kotlin", "maven", "gradle",
		},
		Optional: []string{
			"scala",
		},
	},

	// Desktop groups
	"gaming": {
		Name:        "gaming",
		Description: "Gaming packages and tools",
		Packages: []string{
			"steam", "wine", "gamemode", "mangohud", "lutris",
		},
	},
	"multimedia": {
		Name:        "multimedia",
		Description: "Multimedia applications",
		Packages: []string{
			"obs-studio", "kdenlive", "audacity", "gimp", "inkscape",
			"vlc", "mpv", "ffmpeg",
		},
	},
	"office": {
		Name:        "office",
		Description: "Office productivity applications",
		Packages: []string{
			"libreoffice-fresh", "thunderbird", "zathura", "evince", "okular",
		},
	},
	"virtualization": {
		Name:        "virtualization",
		Description: "Virtualization tools",
		Packages: []string{
			"qemu-full", "libvirt", "virt-manager", "docker", "docker-compose",
		},
	},
	"fonts": {
		Name:        "fonts",
		Description: "Essential fonts",
		Packages: []string{
			"ttf-dejavu", "ttf-liberation", "noto-fonts",
			"noto-fonts-emoji", "ttf-fira-code", "ttf-jetbrains-mono",
		},
	},

	// ShedOS desktop environments
	"shedos-hyprland": {
		Name:        "shedos-hyprland",
		Description: "ShedOS with Hyprland compositor",
		Packages: []string{
			"hyprland", "waybar", "walker", "kitty",
			"swww", "hyprlock", "hypridle", "grim", "slurp",
			"wl-clipboard", "xdg-desktop-portal-hyprland",
			"polkit-kde-agent", "qt5-wayland", "qt6-wayland",
			"hyprsunset", "xdg-desktop-portal-gtk", "xdg-utils",
			"xdg-user-dirs", "sddm", "hyprpicker",
		},
	},
	"shedos-gnome": {
		Name:        "shedos-gnome",
		Description: "ShedOS with GNOME desktop",
		Packages: []string{
			"gnome", "gnome-circle",
		},
	},
	"shedos-kde": {
		Name:        "shedos-kde",
		Description: "ShedOS with KDE Plasma desktop",
		Packages: []string{
			"plasma", "kde-applications", "sddm",
		},
	},
	"shedos-cosmic": {
		Name:        "shedos-cosmic",
		Description: "ShedOS with COSMIC desktop (System76)",
		Packages: []string{
			"cosmic",
		},
	},
	"shedos-budgie": {
		Name:        "shedos-budgie",
		Description: "ShedOS with Budgie desktop",
		Packages: []string{
			"budgie",
		},
	},
	"shedos-cinnamon": {
		Name:        "shedos-cinnamon",
		Description: "ShedOS with Cinnamon desktop",
		Packages: []string{
			"cinnamon",
		},
	},
	"shedos-deepin": {
		Name:        "shedos-deepin",
		Description: "ShedOS with Deepin desktop",
		Packages: []string{
			"deepin", "deepin-kwin", "deepin-extra",
		},
	},
	"shedos-mate": {
		Name:        "shedos-mate",
		Description: "ShedOS with MATE desktop",
		Packages: []string{
			"mate", "mate-extra",
		},
	},
}

// GroupsConfig represents the groups.toml configuration file
type GroupsConfig struct {
	Groups map[string]GroupConfigEntry `toml:"groups"`
}

// GroupConfigEntry is a single group entry in config
type GroupConfigEntry struct {
	Description string   `toml:"description"`
	Packages    []string `toml:"packages"`
	Optional    []string `toml:"optional"`
	Includes    []string `toml:"includes"`
}

// GroupRegistry manages package groups
type GroupRegistry struct {
	groups map[string]PackageGroup
}

// NewGroupRegistry creates a registry with default groups
func NewGroupRegistry() *GroupRegistry {
	r := &GroupRegistry{
		groups: make(map[string]PackageGroup),
	}
	// Copy default groups
	for name, group := range DefaultGroups {
		r.groups[name] = group
	}
	return r
}

// NewGroupRegistryWithConfig creates a registry and loads custom groups
func NewGroupRegistryWithConfig(cfg *config.Config) *GroupRegistry {
	r := NewGroupRegistry()

	// Try to load custom groups from default config directory
	home, _ := os.UserHomeDir()
	groupsFile := filepath.Join(home, ".config", "shedman", "groups.toml")

	if err := r.LoadFromFile(groupsFile); err != nil {
		// Silent failure - config file may not exist
	}

	return r
}

// LoadFromFile loads custom groups from a TOML file
func (r *GroupRegistry) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var groupsCfg GroupsConfig
	if err := toml.Unmarshal(data, &groupsCfg); err != nil {
		return fmt.Errorf("failed to parse groups.toml: %w", err)
	}

	for name, entry := range groupsCfg.Groups {
		r.groups[name] = PackageGroup{
			Name:        name,
			Description: entry.Description,
			Packages:    entry.Packages,
			Optional:    entry.Optional,
			Includes:    entry.Includes,
		}
	}

	return nil
}

// GetGroup returns a group by name (with or without @ prefix)
func (r *GroupRegistry) GetGroup(name string) (*PackageGroup, bool) {
	name = strings.TrimPrefix(name, GroupPrefix)
	group, exists := r.groups[name]
	if !exists {
		return nil, false
	}
	return &group, true
}

// ListGroups returns all group names sorted
func (r *GroupRegistry) ListGroups() []string {
	names := make([]string, 0, len(r.groups))
	for name := range r.groups {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ListGroupsFormatted returns a formatted list for display
func (r *GroupRegistry) ListGroupsFormatted() []string {
	names := r.ListGroups()
	result := make([]string, 0, len(names))

	for _, name := range names {
		group := r.groups[name]
		pkgCount := len(group.Packages)
		result = append(result, fmt.Sprintf("@%-20s %3d packages  %s", name, pkgCount, group.Description))
	}

	return result
}

// RegisterGroup adds or updates a group
func (r *GroupRegistry) RegisterGroup(group PackageGroup) {
	r.groups[group.Name] = group
}

// IsGroupReference checks if a string is a group reference (@name)
func IsGroupReference(s string) bool {
	return strings.HasPrefix(s, GroupPrefix)
}

// ExpandGroups expands group references to package lists with nested support
func (r *GroupRegistry) ExpandGroups(packages []string) ([]string, error) {
	result := make([]string, 0)
	seen := make(map[string]bool)
	visited := make(map[string]bool) // For cycle detection

	for _, pkg := range packages {
		if IsGroupReference(pkg) {
			expanded, err := r.expandGroupRecursive(pkg, seen, visited)
			if err != nil {
				return nil, err
			}
			result = append(result, expanded...)
		} else {
			if !seen[pkg] {
				result = append(result, pkg)
				seen[pkg] = true
			}
		}
	}

	return result, nil
}

// expandGroupRecursive expands a group with nested support and cycle detection
func (r *GroupRegistry) expandGroupRecursive(name string, seen, visited map[string]bool) ([]string, error) {
	name = strings.TrimPrefix(name, GroupPrefix)

	// Cycle detection
	if visited[name] {
		return nil, fmt.Errorf("cycle detected in group: @%s", name)
	}
	visited[name] = true
	defer func() { visited[name] = false }()

	group, exists := r.GetGroup(name)
	if !exists {
		return nil, fmt.Errorf("unknown group: @%s", name)
	}

	result := make([]string, 0)

	// First expand included groups
	for _, include := range group.Includes {
		expanded, err := r.expandGroupRecursive(include, seen, visited)
		if err != nil {
			return nil, err
		}
		result = append(result, expanded...)
	}

	// Then add this group's packages
	for _, p := range group.Packages {
		if !seen[p] {
			result = append(result, p)
			seen[p] = true
		}
	}

	return result, nil
}

// ExpandGroupsWithOptional expands groups including optional packages
func (r *GroupRegistry) ExpandGroupsWithOptional(packages []string, includeOptional bool) ([]string, []string, error) {
	required := make([]string, 0)
	optional := make([]string, 0)
	seen := make(map[string]bool)
	visited := make(map[string]bool)

	for _, pkg := range packages {
		if IsGroupReference(pkg) {
			req, opt, err := r.expandGroupWithOptionalRecursive(pkg, seen, visited, includeOptional)
			if err != nil {
				return nil, nil, err
			}
			required = append(required, req...)
			optional = append(optional, opt...)
		} else {
			if !seen[pkg] {
				required = append(required, pkg)
				seen[pkg] = true
			}
		}
	}

	return required, optional, nil
}

// expandGroupWithOptionalRecursive handles nested expansion with optional packages
func (r *GroupRegistry) expandGroupWithOptionalRecursive(name string, seen, visited map[string]bool, includeOptional bool) ([]string, []string, error) {
	name = strings.TrimPrefix(name, GroupPrefix)

	if visited[name] {
		return nil, nil, fmt.Errorf("cycle detected in group: @%s", name)
	}
	visited[name] = true
	defer func() { visited[name] = false }()

	group, exists := r.GetGroup(name)
	if !exists {
		return nil, nil, fmt.Errorf("unknown group: @%s", name)
	}

	required := make([]string, 0)
	optional := make([]string, 0)

	// Expand included groups first
	for _, include := range group.Includes {
		req, opt, err := r.expandGroupWithOptionalRecursive(include, seen, visited, includeOptional)
		if err != nil {
			return nil, nil, err
		}
		required = append(required, req...)
		optional = append(optional, opt...)
	}

	// Add packages
	for _, p := range group.Packages {
		if !seen[p] {
			required = append(required, p)
			seen[p] = true
		}
	}

	// Add optional if requested
	if includeOptional {
		for _, p := range group.Optional {
			if !seen[p] {
				optional = append(optional, p)
				seen[p] = true
			}
		}
	}

	return required, optional, nil
}

// GetGroupDescription returns a formatted description of a group
func (r *GroupRegistry) GetGroupDescription(name string) string {
	group, exists := r.GetGroup(name)
	if !exists {
		return ""
	}
	return fmt.Sprintf("@%s - %s (%d packages)", group.Name, group.Description, len(group.Packages))
}

// HasGroup checks if a group exists
func (r *GroupRegistry) HasGroup(name string) bool {
	_, exists := r.GetGroup(name)
	return exists
}
