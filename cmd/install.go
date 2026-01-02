package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/pkg/shedman/installer"
	"github.com/theshedman/shedman/pkg/shedman/output"
	"github.com/theshedman/shedman/pkg/shedman/pkgdb"
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
		// Build package list
		pkgs := make([]pkgdb.PackageInfo, len(args))
		for i, arg := range args {
			pkgs[i] = pkgdb.PackageInfo{
				Name:   arg,
				Source: pkgdb.SourceOfficial,
			}
		}

		// Show what we're installing
		if !quietFlag {
			output.Info("Installing %d package(s)...", len(pkgs))
			for _, pkg := range pkgs {
				fmt.Printf("  → %s\n", pkg.Name)
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
			pi := installer.NewPacmanInstaller()
			names := make([]string, len(pkgs))
			for i, p := range pkgs {
				names[i] = p.Name
			}
			cmdArgs := pi.BuildCommand(names, opts)
			fmt.Printf("  %v\n", cmdArgs)
			return nil
		}

		// Execute installation
		pi := installer.NewPacmanInstaller()
		if err := pi.InstallMultiple(pkgs, opts); err != nil {
			output.Error("Installation failed: %v", err)
			return err
		}

		if !quietFlag {
			output.Success("Installation complete.")
		}

		return nil
	},
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