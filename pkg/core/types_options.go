package core

// InstallOptions for package installation
type InstallOptions struct {
	Needed       bool   // Skip if already installed
	AsDeps       bool   // Mark as dependency
	AsExplicit   bool   // Mark as explicitly installed
	NoConfirm    bool   // Non-interactive mode
	DownloadOnly bool   // Download but don't install
	Overwrite    string // Overwrite glob pattern (pacman-specific)
}

// RemoveOptions for package removal
type RemoveOptions struct {
	Cascade   bool // Remove dependents
	NoSave    bool // Don't save config files
	Recursive bool // Remove unused dependencies
	NoConfirm bool // Non-interactive mode
}

// UpgradeOptions for package upgrades
type UpgradeOptions struct {
	NoConfirm      bool     // Non-interactive mode
	Refresh        bool     // Refresh database first
	Needed         bool     // Skip packages that are already up-to-date
	IgnorePkgs     []string // Packages to ignore during upgrade
	IgnoreGroups   []string // Groups to ignore during upgrade
	TargetBackends []string // Specific backends to target (empty = all)
	Delta          bool     // Enable delta updates
	LimitRate      string   // Download rate limit (e.g., 500K, 2M)
	Retry          int      // Download retry count
	Timeout        int      // Network timeout in seconds
}

// DefaultInstallOptions returns sensible defaults
func DefaultInstallOptions() InstallOptions {
	return InstallOptions{
		Needed:    true,
		NoConfirm: false,
	}
}

// DefaultRemoveOptions returns sensible defaults
func DefaultRemoveOptions() RemoveOptions {
	return RemoveOptions{
		Recursive: true,
		NoConfirm: false,
	}
}

// DefaultUpgradeOptions returns sensible defaults
func DefaultUpgradeOptions() UpgradeOptions {
	return UpgradeOptions{
		Refresh:   true,
		NoConfirm: false,
	}
}

// WithNeeded sets the Needed flag
func (o InstallOptions) WithNeeded(needed bool) InstallOptions {
	o.Needed = needed
	return o
}

// WithNoConfirm sets the NoConfirm flag
func (o InstallOptions) WithNoConfirm(noConfirm bool) InstallOptions {
	o.NoConfirm = noConfirm
	return o
}

// WithAsDeps sets the AsDeps flag
func (o InstallOptions) WithAsDeps(asDeps bool) InstallOptions {
	o.AsDeps = asDeps
	return o
}

// WithNeeded sets the Needed flag for UpgradeOptions
func (o UpgradeOptions) WithNeeded(needed bool) UpgradeOptions {
	o.Needed = needed
	return o
}

// WithNoConfirm sets the NoConfirm flag for UpgradeOptions
func (o UpgradeOptions) WithNoConfirm(noConfirm bool) UpgradeOptions {
	o.NoConfirm = noConfirm
	return o
}

// WithRefresh sets the Refresh flag for UpgradeOptions
func (o UpgradeOptions) WithRefresh(refresh bool) UpgradeOptions {
	o.Refresh = refresh
	return o
}

// WithCascade sets the Cascade flag for RemoveOptions
func (o RemoveOptions) WithCascade(cascade bool) RemoveOptions {
	o.Cascade = cascade
	return o
}

// WithNoSave sets the NoSave flag for RemoveOptions
func (o RemoveOptions) WithNoSave(noSave bool) RemoveOptions {
	o.NoSave = noSave
	return o
}

// WithRecursive sets the Recursive flag for RemoveOptions
func (o RemoveOptions) WithRecursive(recursive bool) RemoveOptions {
	o.Recursive = recursive
	return o
}

// WithNoConfirm sets the NoConfirm flag for RemoveOptions
func (o RemoveOptions) WithNoConfirm(noConfirm bool) RemoveOptions {
	o.NoConfirm = noConfirm
	return o
}
