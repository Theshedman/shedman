package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/internal/output"
	"github.com/theshedman/shedman/pkg/core"
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

var InstallCmd = &cobra.Command{
	Use:   "install [packages...]",
	Short: "Install packages",
	Long:  `Install packages from configured repositories (Official, AUR, ShedOS).`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration
		cfg, err := config.LoadDefault()
		if err != nil {
			output.Warning("Failed to load config, using defaults: %v", err)
			cfg = config.Default()
		}

		// Determine source based on flags
		source := determineSource()

		// Query packages from appropriate database
		backend, err := selectDatabase(source, cfg)
		if err != nil {
			return fmt.Errorf("failed to initialize backend: %w", err)
		}

		eng := core.NewEngineWithBackend(backend)
		return RunInstall(eng, cmd, args, os.Stdout, cfg)
	},
}

// RunInstall executes the install command logic
func RunInstall(eng *core.Engine, cmd *cobra.Command, args []string, w io.Writer, cfg *config.Config) error {
	backend := eng.GetOfficialBackend()
	if backend == nil {
		return core.ErrBackendNotFound
	}

	// Parse package requests
	var pkgs []core.PackageInfo
	for _, arg := range args {
		// Handle groups
		if len(arg) > 0 && arg[0] == '@' {
			output.Info("Resolving group %s...", arg)
			registry := core.NewGroupRegistryWithConfig(cfg)
			expanded, err := registry.ExpandGroups([]string{arg})
			if err != nil {
				output.Error("Failed to expand group %s: %v", arg, err)
				continue
			}

			for _, p := range expanded {
				// Add to args to be processed (simple way) or handle directly
				// Helper: process expanded packages
				// Fetch package info
				info, err := backend.Info(p)
				if err != nil {
					output.Error("Failed to query package %s (from group %s): %v", p, arg, err)
					continue
				}
				if info == nil {
					output.Error("Package %s (from group %s) not found", p, arg)
					continue
				}
				pkgs = append(pkgs, *info)
			}
			continue
		}

		// Parse package spec (name@version)
		req := core.ParsePackageRequest(arg)
		pkgName := req.Name

		// Look up package info
		info, err := backend.Info(pkgName)
		if err != nil {
			output.Error("Failed to query package %s: %v", pkgName, err)
			continue
		}
		if info == nil {
			output.Error("Package not found: %s", pkgName)
			continue
		}

		// If version requested, check match (simplified logic)
		if req.Version != "" && info.Version != req.Version {
			output.Warning("Requested version %s but found %s", req.Version, info.Version)
			output.Warning("Requested version %s but found %s", req.Version, info.Version)
			// Continue with found version
		}

		pkgs = append(pkgs, *info)
	}

	if len(pkgs) == 0 {
		return fmt.Errorf("no packages to install")
	}

	// Show what we're installing using summary table
	quiet, _ := cmd.Flags().GetBool("quiet")
	yes, _ := cmd.Flags().GetBool("yes")
	if !quiet {
		summary := output.NewInstallSummaryTable()
		for _, pkg := range pkgs {
			status := "install"
			if backend.IsInstalled(pkg.Name) {
				status = "reinstall"
			}
			summary.AddPackage(output.SummaryRow{
				Name:    pkg.Name,
				Version: pkg.Version,
				Source:  string(pkg.Source),
				Size:    pkg.Size,
				Status:  status,
			})
		}
		summary.Print()

		// Confirmation prompt
		if !yes && cfg.General.Confirm {
			if !output.Confirm("Proceed with installation?", output.ConfirmOptions{Default: true}) {
				return fmt.Errorf("installation cancelled")
			}
		}
	}

	// Build install options
	opts := core.InstallOptions{
		Needed:       installNeeded,
		AsDeps:       installAsDeps,
		AsExplicit:   installAsExplicit || (!installAsDeps),
		NoConfirm:    yes,
		DownloadOnly: installDownloadOnly,
		Overwrite:    installOverwrite,
	}

	// Dry-run mode
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		cmd.Println("\nDry-run mode - would execute:")
		for _, pkg := range pkgs {
			cmd.Printf("  Install %s from %s\n", pkg.Name, pkg.Source)
		}
		return nil
	}

	// Execute installation based on source
	if err := executeInstall(backend, cfg, pkgs, opts); err != nil {
		return err
	}

	if !quiet {
		fmt.Fprintln(w, "Installation complete.")
	}

	// Post-install: Handle configuration management
	handlePostInstall(eng, pkgs, w)

	return nil
}

func handlePostInstall(eng *core.Engine, pkgs []core.PackageInfo, w io.Writer) {
	// Check for configuration updates
	fmt.Fprintln(w, "Checking for configuration updates...")

	backend := eng.GetOfficialBackend()
	engine := CreateConfigEngine()

	for _, pkg := range pkgs {
		if fp, ok := backend.(core.FileProvider); ok {
			files, err := fp.GetPackageFiles(pkg.Name)
			if err != nil {
				continue
			}

			for _, file := range files {
				absPath := file
				if len(absPath) > 0 && absPath[0] != '/' {
					absPath = "/" + absPath
				}
				pacnewPath := absPath + ".pacnew"
				if _, err := os.Stat(pacnewPath); err == nil {
					fmt.Fprintf(w, "Processing config: %s\n", absPath)
					if err := engine.Apply(pkg.Name, pacnewPath, absPath); err != nil {
						fmt.Fprintf(w, "Failed to apply config for %s: %v\n", absPath, err)
					}
				}
			}
		}
	}
}

// determineSource returns the forced source based on flags, or empty for auto
func determineSource() string {
	if installFromAUR {
		return "aur"
	}
	if installFromOfficial {
		return "official"
	}
	if installFromShedOS {
		return "shedos"
	}
	return "" // Auto-detect
}

// selectDatabase returns the appropriate backend based on source
func selectDatabase(source string, cfg *config.Config) (core.OfficialBackend, error) {
	// Verify backend is available

	// Use factory to get main backend
	backend, err := DetectBackendWithConfig(nil)
	if err != nil {
		return nil, err
	}
	return backend, nil
}

// executeInstall runs the appropriate installer for each package
func executeInstall(pacmanBackend core.OfficialBackend, cfg *config.Config, pkgs []core.PackageInfo, opts core.InstallOptions) error {
	// Group packages by source
	var official []core.PackageInfo
	var aur []core.PackageInfo
	var shedos []core.PackageInfo

	for _, pkg := range pkgs {
		switch pkg.Source {
		case core.SourceAUR:
			aur = append(aur, pkg)
		default:
			official = append(official, pkg)
		}
	}

	// Check if pacman backend required
	needsPacman := len(official) > 0 || len(shedos) > 0

	if needsPacman {
		if pacmanBackend == nil {
			return fmt.Errorf("pacman backend required but not provided")
		}
	}

	// Build backend install options
	backendOpts := core.InstallOptions{
		Needed:       opts.Needed,
		AsDeps:       opts.AsDeps,
		AsExplicit:   opts.AsExplicit,
		NoConfirm:    opts.NoConfirm,
		DownloadOnly: opts.DownloadOnly,
		Overwrite:    opts.Overwrite,
	}

	// Install official packages
	if len(official) > 0 {
		var pkgNames []string
		for _, pkg := range official {
			pkgNames = append(pkgNames, pkg.Name)
		}
		if err := pacmanBackend.Install(pkgNames, backendOpts); err != nil {
			return fmt.Errorf("pacman install failed: %w", err)
		}
	}

	// Install AUR packages
	if len(aur) > 0 {
		// Use factory
		ai := CreateAURInstaller(cfg)
		for _, pkg := range aur {
			// Install AUR package using helper
			// Note: AUR installer handles dependency resolution internally
			if err := ai.Install(pkg.Name); err != nil {
				return fmt.Errorf("AUR install failed for %s: %w", pkg.Name, err)
			}
		}
	}

	return nil
}

func init() {
	InstallCmd.Flags().BoolVar(&installNeeded, "needed", false, "Do not reinstall up-to-date packages")
	InstallCmd.Flags().BoolVar(&installAsDeps, "asdeps", false, "Install as dependency")
	InstallCmd.Flags().BoolVar(&installAsExplicit, "asexplicit", false, "Install as explicit")
	InstallCmd.Flags().BoolVar(&installDownloadOnly, "download-only", false, "Download without installing")
	InstallCmd.Flags().StringVar(&installOverwrite, "overwrite", "", "Overwrite conflicting files (glob)")
	InstallCmd.Flags().BoolVar(&installFromAUR, "aur", false, "Force install from AUR")
	InstallCmd.Flags().BoolVar(&installFromOfficial, "official", false, "Force install from official repos")
	InstallCmd.Flags().BoolVar(&installFromShedOS, "shedos", false, "Force install from ShedOS repo")
}
