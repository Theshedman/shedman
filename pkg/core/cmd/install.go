package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/internal/output"
	"github.com/theshedman/shedman/pkg/core"
	shedrepo "github.com/theshedman/shedman/pkg/core/providers/shed"
	"github.com/theshedman/shedman/pkg/svc"
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

		backend, err := DetectBackendWithConfig(&cfg.Backend)
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

	forceSource, err := determineSource(flags)
	if err != nil {
		return err
	}

	resolver, err := buildInstallResolver(cfg, backend, forceSource)
	if err != nil {
		return err
	}

	var pkgs []core.PackageInfo
	var localFiles []string

	for _, arg := range args {
		if isLocalPackage(arg) {
			localFiles = append(localFiles, arg)
			continue
		}

		req := core.ParsePackageRequest(arg)
		if req.IsGroup {
			groupName := "@" + req.Name
			_, _ = fmt.Fprintf(w, "Resolving group %s...\n", groupName)

			registry := core.NewGroupRegistryWithConfig(cfg)
			expanded, err := registry.ExpandGroups([]string{groupName})
			if err != nil {
				_, _ = fmt.Fprintf(w, "Failed to expand group %s: %v\n", groupName, err)
				continue
			}

			for _, p := range expanded {
				info, err := resolver.FindPackage(p, forceSource)
				if err != nil {
					_, _ = fmt.Fprintf(w, "Failed to query package %s (from group %s): %v\n", p, groupName, err)
					continue
				}
				if info == nil {
					_, _ = fmt.Fprintf(w, "Package %s (from group %s) not found\n", p, groupName)
					continue
				}
				pkgs = append(pkgs, *info)
			}
			continue
		}

		info, err := resolver.FindPackage(req.Name, forceSource)
		if err != nil {
			_, _ = fmt.Fprintf(w, "Failed to query package %s: %v\n", req.Name, err)
			continue
		}
		if info == nil {
			_, _ = fmt.Fprintf(w, "Package not found: %s\n", req.Name)
			continue
		}
		if req.Version != "" && info.Version != req.Version {
			_, _ = fmt.Fprintf(w, "Warning: Requested version %s but found %s\n", req.Version, info.Version)
		}
		pkgs = append(pkgs, *info)
	}

	if len(pkgs) == 0 && len(localFiles) == 0 {
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
		for _, path := range localFiles {
			size := int64(0)
			if info, err := os.Stat(path); err == nil {
				size = info.Size()
			}
			summary.AddPackage(output.SummaryRow{
				Name:    filepath.Base(path),
				Version: "-",
				Source:  "local",
				Size:    size,
				Status:  "install",
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
		NoConfirm:    flags.Yes || !cfg.General.Confirm,
		DownloadOnly: flags.DownloadOnly,
		Overwrite:    flags.Overwrite,
	}

	if flags.DryRun {
		_, _ = fmt.Fprintln(w, "\nDry-run mode - would execute:")

		for _, pkg := range pkgs {
			_, _ = fmt.Fprintf(w, "  Install %s from %s\n", pkg.Name, pkg.Source)
		}
		for _, path := range localFiles {
			_, _ = fmt.Fprintf(w, "  Install local package %s\n", path)
		}
		return nil
	}

	// Execute installation
	if err := executeInstall(backend, cfg, pkgs, opts, w); err != nil {
		return err
	}
	if len(localFiles) > 0 {
		if opts.DownloadOnly {
			return fmt.Errorf("download-only is not supported for local packages")
		}
		for _, path := range localFiles {
			if err := eng.InstallFile(path, opts); err != nil {
				return err
			}
		}
	}

	if !flags.Quiet {
		_, _ = fmt.Fprintln(w, "Installation complete.")

	}

	// Post-install: Handle configuration management
	handlePostInstall(eng, pkgs, w, flags.Yes || !cfg.General.Confirm)

	return nil
}

func handlePostInstall(eng *core.Engine, pkgs []core.PackageInfo, w io.Writer, skipServicePrompt bool) {
	// Check for configuration updates
	backend := eng.GetOfficialBackend()
	if backend == nil {
		return
	}

	engine := CreateConfigEngine()
	var services []string

	for _, pkg := range pkgs {
		if fp, ok := backend.(core.FileProvider); ok {
			files, err := fp.GetPackageFiles(pkg.Name)
			if err != nil {
				continue
			}

			services = append(services, detectSystemdUnits(files)...)

			for _, file := range files {
				absPath := file
				if len(absPath) > 0 && absPath[0] != '/' {
					absPath = "/" + absPath
				}
				pacnewPath := absPath + ".pacnew"
				if _, err := os.Stat(pacnewPath); err == nil {
					_, _ = fmt.Fprintf(w, "Processing config: %s\n", absPath)

					if err := engine.Apply(pkg.Name, pacnewPath, absPath); err != nil {
						_, _ = fmt.Fprintf(w, "Failed to apply config for %s: %v\n", absPath, err)

					}
				}
			}
		}
	}

	if skipServicePrompt {
		return
	}

	services = dedupeStrings(services)
	if len(services) == 0 {
		return
	}

	if err := promptEnableServices(w, svc.New(), services); err != nil {
		_, _ = fmt.Fprintf(w, "Failed to enable services: %v\n", err)
	}
}

// determineSource returns the forced source based on flags, or empty for auto.
func determineSource(flags InstallFlags) (string, error) {
	var sources []string
	if flags.FromAUR {
		sources = append(sources, core.SourceAUR)
	}
	if flags.FromOfficial {
		sources = append(sources, core.SourceOfficial)
	}
	if flags.FromShedOS {
		sources = append(sources, core.SourceShedOS)
	}
	if len(sources) > 1 {
		return "", fmt.Errorf("only one of --aur, --official, or --shedos can be specified")
	}
	if len(sources) == 1 {
		return sources[0], nil
	}
	return "", nil
}

type packageDBAdapter struct {
	search func(string) ([]core.PackageInfo, error)
	info   func(string) (*core.PackageInfo, error)
}

func (a packageDBAdapter) Search(query string) ([]core.PackageInfo, error) {
	return a.search(query)
}

func (a packageDBAdapter) GetInfo(name string) (*core.PackageInfo, error) {
	return a.info(name)
}

func buildInstallResolver(cfg *config.Config, backend core.OfficialBackend, forceSource string) (*core.MultiSourceResolver, error) {
	if cfg == nil {
		cfg = config.Default()
	}
	resolver := core.NewMultiSource()

	if backend != nil {
		resolver.AddSource(core.SourceOfficial, packageDBAdapter{
			search: backend.Search,
			info:   backend.Info,
		})
	} else if forceSource == core.SourceOfficial {
		return nil, core.ErrBackendNotFound
	}

	fsCache := core.NewFileSystemCache()
	timeout := 30 * time.Second
	if cfg.Network.Timeout > 0 {
		timeout = time.Duration(cfg.Network.Timeout) * time.Second
	}
	var shedBackend *shedrepo.Backend
	if len(cfg.Mirrors.ShedOS) > 0 {
		shedBackend = shedrepo.NewWithMirrors(cfg.Mirrors.ShedOS, fsCache, timeout)
	} else {
		shedBackend = shedrepo.New(fsCache, timeout)
	}
	resolver.AddSource(core.SourceShedOS, packageDBAdapter{
		search: shedBackend.Search,
		info:   shedBackend.Info,
	})

	if cfg.AUR.Enabled && core.IsAURAvailable() {
		resolver.AddSource(core.SourceAUR, core.NewAURDBWithConfig(cfg))
	} else if forceSource == core.SourceAUR {
		return nil, core.ErrAURNotAvailable
	}

	return resolver, nil
}

// executeInstall runs the appropriate installer for each package.
func executeInstall(pacmanBackend core.OfficialBackend, cfg *config.Config, pkgs []core.PackageInfo, opts core.InstallOptions, w io.Writer) error {
	// Group packages by source
	var official []core.PackageInfo
	var aur []core.PackageInfo

	for _, pkg := range pkgs {
		switch pkg.Source {
		case core.SourceAUR:
			aur = append(aur, pkg)
		default:
			official = append(official, pkg)
		}
	}

	// Check if pacman backend required
	needsPacman := len(official) > 0

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
			if err := installAURPackage(ai, pkg, opts, cfg, w); err != nil {
				return fmt.Errorf("AUR install failed for %s: %w", pkg.Name, err)
			}
		}
	}

	return nil
}

func installAURPackage(ai *core.AURInstaller, pkg core.PackageInfo, opts core.InstallOptions, cfg *config.Config, w io.Writer) error {
	first := ai.IsFirstTime(pkg.Name)
	if err := ai.Clone(pkg.Name); err != nil {
		return fmt.Errorf("clone failed: %w", err)
	}

	var review string
	var err error
	if first {
		review, err = ai.GetPKGBUILD(pkg.Name)
	} else {
		review, err = ai.GetPKGBUILDDiff(pkg.Name)
	}
	if err != nil {
		return err
	}

	if review != "" {
		title := "PKGBUILD"
		if !first {
			title = "PKGBUILD diff"
		}
		_, _ = fmt.Fprintf(w, "\n%s for %s:\n%s\n", title, pkg.Name, review)
	}

	confirmOpts := output.ConfirmOptions{Default: false}
	if cfg.General.PromptTimeout > 0 {
		confirmOpts.Timeout = time.Duration(cfg.General.PromptTimeout) * time.Second
	}
	if opts.NoConfirm {
		confirmOpts.Default = true
		confirmOpts.SkipPrompt = true
	}
	if !output.Confirm("Proceed with AUR build?", confirmOpts) {
		return fmt.Errorf("AUR install cancelled for %s", pkg.Name)
	}

	aurOpts := core.AUROptionsFromConfig(cfg)
	aurOpts.NoConfirm = opts.NoConfirm

	if aurOpts.FetchPGPKeys {
		if err := ai.FetchPGPKeys(pkg.Name); err != nil {
			_, _ = fmt.Fprintf(w, "Warning: failed to fetch PGP keys for %s: %v\n", pkg.Name, err)
		}
	}
	if aurOpts.VerifyChecksums {
		if err := ai.VerifyChecksumsWithOptions(pkg.Name, aurOpts); err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}
	}

	if opts.DownloadOnly {
		return nil
	}

	if err := ai.BuildWithOptions(pkg.Name, aurOpts); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}
	if err := ai.Install(pkg.Name, aurOpts); err != nil {
		return fmt.Errorf("install failed: %w", err)
	}
	if cfg.AUR.CleanAfterBuild {
		if err := ai.Clean(pkg.Name); err != nil {
			_, _ = fmt.Fprintf(w, "Warning: cleanup failed for %s: %v\n", pkg.Name, err)
		}
	}
	return nil
}

func isLocalPackage(arg string) bool {
	if arg == "" {
		return false
	}
	info, err := os.Stat(arg)
	if err != nil || info.IsDir() {
		return false
	}
	switch {
	case strings.HasSuffix(arg, ".pkg.tar.zst"):
		return true
	case strings.HasSuffix(arg, ".pkg.tar.xz"):
		return true
	case strings.HasSuffix(arg, ".pkg.tar.gz"):
		return true
	default:
		return false
	}
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
