package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/pkg/shedman/backend"
	"github.com/theshedman/shedman/pkg/shedman/backend/pacman"
	"github.com/theshedman/shedman/pkg/shedman/config"
	"github.com/theshedman/shedman/pkg/shedman/installer"
	"github.com/theshedman/shedman/pkg/shedman/output"
	"github.com/theshedman/shedman/pkg/shedman/pkgdb"
	"github.com/theshedman/shedman/pkg/shedman/resolver"
)

var (
	installNeeded       bool
	installAsDeps       bool
	installAsExplicit   bool
	installDownloadOnly bool
	installOverwrite    string
	installFromAUR      bool
	installFromOfficial bool
	installFromShedOS   bool
)

var installCmd = &cobra.Command{
	Use:   "install [packages...]",
	Short: "Install packages",
	Long: `Install packages from configured sources.

Examples:
  shedman install neovim          # Install from best source
  shedman install neovim@0.10.0   # Install specific version
  shedman install neovim --aur    # Force from AUR
  shedman install @dev            # Install package group`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration
		cfg, err := loadConfig()
		if err != nil {
			output.Warning("Failed to load config, using defaults: %v", err)
			cfg = config.Default()
		}

		// Determine source based on flags
		source := determineSource()

		// Check AUR availability if AUR is requested
		if source == pkgdb.SourceAUR && !backend.IsAURAvailable() {
			return backend.ErrAURNotAvailable
		}

		// Query packages from appropriate database
		db := selectDatabase(cfg, source)
		if db == nil {
			return fmt.Errorf("failed to initialize package database")
		}

		// Parse package requests
		var pkgs []pkgdb.PackageInfo
		for _, arg := range args {
			req := resolver.ParseRequest(arg)

			// Handle groups
			if req.IsGroup {
				output.Info("Package groups not yet implemented: %s", arg)
				continue
			}

			// Look up package info
			info, err := db.GetInfo(req.Name)
			if err != nil {
				output.Error("Failed to query package %s: %v", req.Name, err)
				continue
			}
			if info == nil {
				output.Error("Package not found: %s", req.Name)
				continue
			}

			pkgs = append(pkgs, *info)
		}

		if len(pkgs) == 0 {
			return fmt.Errorf("no packages to install")
		}

		// Show what we're installing
		if !quietFlag {
			output.Info("Installing %d package(s)...", len(pkgs))
			for _, pkg := range pkgs {
				fmt.Printf("  → %s [%s]\n", pkg.Name, pkg.Source)
			}
		}

		// Build install options
		opts := installer.Options{
			Needed:       installNeeded,
			AsDeps:       installAsDeps,
			AsExplicit:   installAsExplicit || (!installAsDeps),
			NoConfirm:    yesFlag,
			DownloadOnly: installDownloadOnly,
			Overwrite:    installOverwrite,
		}

		// Dry-run mode
		if dryRunFlag {
			cmd.Println("\nDry-run mode - would execute:")
			for _, pkg := range pkgs {
				cmd.Printf("  Install %s from %s\n", pkg.Name, pkg.Source)
			}
			return nil
		}

		// Execute installation based on source
		if err := executeInstall(cfg, pkgs, opts); err != nil {
			output.Error("Installation failed: %v", err)
			return err
		}

		if !quietFlag {
			output.Success("Installation complete.")
		}

		return nil
	},
}

// loadConfig loads the shedman configuration file
func loadConfig() (*config.Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(home, ".config", "shedman", "config.toml")
	return config.Load(configPath)
}

// determineSource returns the forced source based on flags, or empty for auto
func determineSource() string {
	if installFromAUR {
		return pkgdb.SourceAUR
	}
	if installFromOfficial {
		return pkgdb.SourceOfficial
	}
	if installFromShedOS {
		return pkgdb.SourceShedOS
	}
	return "" // Auto-detect
}

// selectDatabase returns the appropriate package database based on source
func selectDatabase(cfg *config.Config, source string) pkgdb.PackageDB {
	switch source {
	case pkgdb.SourceAUR:
		return pkgdb.NewAURDBWithConfig(cfg)
	case pkgdb.SourceOfficial:
		return createPacmanDB()
	case pkgdb.SourceShedOS:
		return pkgdb.NewShedDBWithConfig(cfg)
	default:
		// Auto-detect: use MultiSourceResolver to query all sources with priority
		return buildMultiSourceResolver(cfg)
	}
}

// createPacmanDB creates a PacmanDB with the pacman backend properly wired
func createPacmanDB() *pkgdb.PacmanDB {
	db := pkgdb.NewPacmanDB()
	if pacmanBackend, err := pacman.New(); err == nil {
		db.SetBackend(pacmanBackend)
	}
	return db
}

// buildMultiSourceResolver creates a resolver that queries all sources in priority order
func buildMultiSourceResolver(cfg *config.Config) *resolver.MultiSourceResolver {
	ms := resolver.NewMultiSource()

	// Add sources in priority order (highest first)
	// ShedOS first (if mirrors are configured)
	if len(cfg.Mirrors.ShedOS) > 0 {
		ms.AddSource(pkgdb.SourceShedOS, pkgdb.NewShedDBWithConfig(cfg))
	}

	// Official repos (always available on Arch-based systems)
	ms.AddSource(pkgdb.SourceOfficial, createPacmanDB())

	// AUR (if enabled in config)
	if cfg.AUR.Enabled {
		ms.AddSource(pkgdb.SourceAUR, pkgdb.NewAURDBWithConfig(cfg))
	}

	return ms
}

// executeInstall runs the appropriate installer for each package
func executeInstall(cfg *config.Config, pkgs []pkgdb.PackageInfo, opts installer.Options) error {
	// Group packages by source
	official := make([]pkgdb.PackageInfo, 0)
	aur := make([]pkgdb.PackageInfo, 0)
	shedos := make([]pkgdb.PackageInfo, 0)
	shedPkgs := make([]pkgdb.PackageInfo, 0) // .shed format packages

	for _, pkg := range pkgs {
		switch pkg.Source {
		case pkgdb.SourceAUR:
			aur = append(aur, pkg)
		case pkgdb.SourceShedOS:
			// Separate .shed packages from pacman-format ShedOS packages
			if strings.HasSuffix(pkg.Name, ".shed") {
				shedPkgs = append(shedPkgs, pkg)
			} else {
				shedos = append(shedos, pkg)
			}
		default:
			official = append(official, pkg)
		}
	}

	// Determine if we need pacman backend
	needsPacman := len(official) > 0 || len(shedos) > 0
	var pacmanBackend backend.OfficialBackend

	if needsPacman {
		var err error
		pacmanBackend, err = pacman.New()
		if err != nil {
			return fmt.Errorf("pacman backend not available: %w", err)
		}
	}

	// Build backend install options
	backendOpts := backend.InstallOptions{
		Needed:       opts.Needed,
		AsDeps:       opts.AsDeps,
		AsExplicit:   opts.AsExplicit,
		NoConfirm:    opts.NoConfirm,
		DownloadOnly: opts.DownloadOnly,
		Overwrite:    opts.Overwrite,
	}

	// Install official packages
	if len(official) > 0 {
		pkgNames := make([]string, len(official))
		for i, pkg := range official {
			pkgNames[i] = pkg.Name
		}

		if err := pacmanBackend.Install(pkgNames, backendOpts); err != nil {
			return fmt.Errorf("pacman install failed: %w", err)
		}
	}

	// Install AUR packages
	if len(aur) > 0 {
		if !backend.IsAURAvailable() {
			return fmt.Errorf("cannot install AUR packages: %w", backend.ErrAURNotAvailable)
		}
		ai := installer.NewAURInstallerWithConfig(cfg)
		for _, pkg := range aur {
			output.Info("Building AUR package: %s", pkg.Name)

			// Clone or update
			if err := ai.Clone(pkg.Name); err != nil {
				return fmt.Errorf("failed to clone %s: %w", pkg.Name, err)
			}

			// Show PKGBUILD for review
			if ai.IsFirstTime(pkg.Name) {
				pkgbuild, err := ai.GetPKGBUILD(pkg.Name)
				if err == nil {
					output.Info("PKGBUILD for %s:", pkg.Name)
					fmt.Println(pkgbuild)
				}
			}

			// Build
			if err := ai.Build(pkg.Name); err != nil {
				return fmt.Errorf("failed to build %s: %w", pkg.Name, err)
			}

			// Install
			if err := ai.Install(pkg.Name); err != nil {
				return fmt.Errorf("failed to install %s: %w", pkg.Name, err)
			}
		}
	}

	// Install ShedOS packages that use pacman format
	if len(shedos) > 0 {
		pkgNames := make([]string, len(shedos))
		for i, pkg := range shedos {
			pkgNames[i] = pkg.Name
		}

		if err := pacmanBackend.Install(pkgNames, backendOpts); err != nil {
			return fmt.Errorf("failed to install ShedOS packages: %w", err)
		}
	}

	// Install .shed format packages
	if len(shedPkgs) > 0 {
		si := installer.NewShedInstaller()
		for _, pkg := range shedPkgs {
			if err := si.Install(pkg.Name); err != nil {
				return fmt.Errorf("failed to install shed package %s: %w", pkg.Name, err)
			}
		}
	}

	return nil
}

// GetInstallCmd returns the install command for testing
func GetInstallCmd() *cobra.Command {
	return installCmd
}

func init() {
	installCmd.Flags().BoolVar(&installNeeded, "needed", false, "Skip if already installed")
	installCmd.Flags().BoolVar(&installAsDeps, "asdeps", false, "Install as dependency")
	installCmd.Flags().BoolVar(&installAsExplicit, "asexplicit", false, "Install as explicit")
	installCmd.Flags().BoolVar(&installDownloadOnly, "downloadonly", false, "Download without installing")
	installCmd.Flags().StringVar(&installOverwrite, "overwrite", "", "Overwrite conflicting files")
	installCmd.Flags().BoolVar(&installFromAUR, "aur", false, "Force install from AUR")
	installCmd.Flags().BoolVar(&installFromOfficial, "official", false, "Force install from official repos")
	installCmd.Flags().BoolVar(&installFromShedOS, "shedos", false, "Force install from ShedOS repo")

	rootCmd.AddCommand(installCmd)
}
