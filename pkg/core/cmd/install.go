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

		// Populate flags
		flags := InstallFlags{
			Needed:       installNeeded,
			AsDeps:       installAsDeps,
			AsExplicit:   installAsExplicit,
			DownloadOnly: installDownloadOnly,
			Overwrite:    installOverwrite,
			FromAUR:      installFromAUR,
			FromOfficial: installFromOfficial,
			FromShedOS:   installFromShedOS,
		}
		flags.Quiet, _ = cmd.Flags().GetBool("quiet")
		flags.Yes, _ = cmd.Flags().GetBool("yes")
		flags.DryRun, _ = cmd.Flags().GetBool("dry-run")

		source := determineSource(flags)

		backend, err := selectDatabase(source, cfg)
		if err != nil {
			return fmt.Errorf("failed to initialize backend: %w", err)
		}

		eng := core.NewEngineWithBackend(backend)
		return RunInstall(eng, args, flags, cmd.OutOrStdout(), cfg)
	},
}

// InstallFlags holds command-line flags for install
type InstallFlags struct {
	Needed       bool
	AsDeps       bool
	AsExplicit   bool
	DownloadOnly bool
	Overwrite    string
	FromAUR      bool
	FromOfficial bool
	FromShedOS   bool
	Quiet        bool
	Yes          bool
	DryRun       bool
}

func RunInstall(eng *core.Engine, args []string, flags InstallFlags, w io.Writer, cfg *config.Config) error {
	backend := eng.GetOfficialBackend()
	if backend == nil {
		return core.ErrBackendNotFound
	}

	var pkgs []core.PackageInfo
	for _, arg := range args {
		// Handle groups
		if len(arg) > 0 && arg[0] == '@' {
			fmt.Fprintf(w, "Resolving group %s...\n", arg)
			registry := core.NewGroupRegistryWithConfig(cfg)
			expanded, err := registry.ExpandGroups([]string{arg})
			if err != nil {
				fmt.Fprintf(w, "Failed to expand group %s: %v\n", arg, err)
				continue
			}

			for _, p := range expanded {

				// Fetch package info
				info, err := backend.Info(p)
				if err != nil {
					fmt.Fprintf(w, "Failed to query package %s (from group %s): %v\n", p, arg, err)
					continue
				}
				if info == nil {
					fmt.Fprintf(w, "Package %s (from group %s) not found\n", p, arg)
					continue
				}
				pkgs = append(pkgs, *info)
			}
			continue
		}

		req := core.ParsePackageRequest(arg)
		pkgName := req.Name

		info, err := backend.Info(pkgName)
		if err != nil {
			fmt.Fprintf(w, "Failed to query package %s: %v\n", pkgName, err)
			continue
		}
		if info == nil {
			fmt.Fprintf(w, "Package not found: %s\n", pkgName)
			continue
		}

		if req.Version != "" && info.Version != req.Version {
			fmt.Fprintf(w, "Warning: Requested version %s but found %s\n", req.Version, info.Version)
		}

		pkgs = append(pkgs, *info)
	}

	if len(pkgs) == 0 {
		return fmt.Errorf("no packages to install")
	}

	// Show what we're installing using summary table
	if !flags.Quiet {
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
		if !flags.Yes && cfg.General.Confirm {
			if !output.Confirm("Proceed with installation?", output.ConfirmOptions{Default: true}) {
				return fmt.Errorf("installation cancelled")
			}
		}
	}

	opts := core.InstallOptions{
		Needed:       flags.Needed,
		AsDeps:       flags.AsDeps,
		AsExplicit:   flags.AsExplicit || (!flags.AsDeps),
		NoConfirm:    flags.Yes,
		DownloadOnly: flags.DownloadOnly,
		Overwrite:    flags.Overwrite,
	}

	if flags.DryRun {
		fmt.Fprintln(w, "\nDry-run mode - would execute:")
		for _, pkg := range pkgs {
			fmt.Fprintf(w, "  Install %s from %s\n", pkg.Name, pkg.Source)
		}
		return nil
	}

	// Execute installation based on source
	if err := executeInstall(backend, cfg, pkgs, opts); err != nil {
		return err
	}

	if !flags.Quiet {
		fmt.Fprintln(w, "Installation complete.")
	}

	// Post-install: Handle configuration management
	handlePostInstall(eng, pkgs, w)

	return nil
}

func handlePostInstall(eng *core.Engine, pkgs []core.PackageInfo, w io.Writer) {
	// Check for configuration updates
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
func determineSource(flags InstallFlags) string {
	if flags.FromAUR {
		return "aur"
	}
	if flags.FromOfficial {
		return "official"
	}
	if flags.FromShedOS {
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
		ai := CreateAURInstaller(cfg)
		for _, pkg := range aur {
			opts := core.AUROptionsFromConfig(cfg)
			if err := ai.Install(pkg.Name, opts); err != nil {
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
