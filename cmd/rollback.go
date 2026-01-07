package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Jguer/go-alpm/v2"
	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/pkg/shedman"
	"github.com/theshedman/shedman/pkg/shedman/backend"
	"github.com/theshedman/shedman/pkg/shedman/cache"
	"github.com/theshedman/shedman/pkg/shedman/config"
	"github.com/theshedman/shedman/pkg/shedman/output"
)

var (
	rollbackList bool
	rollbackYes  bool
)

func NewRollbackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rollback <package>",
		Short: "Downgrade a package to a previous version",
		Long: `Downgrade a package to a previous version from the local cache.
shedman automatically scans the package cache for older versions of the specified package.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRollback(args[0])
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&rollbackList, "list", "l", false, "List available versions in cache")
	flags.BoolVarP(&rollbackYes, "yes", "y", false, "Skip confirmation")

	return cmd
}

func runRollback(target string) error {
	// target can be "package" or "package@version"
	var pkgName, targetVer string
	if strings.Contains(target, "@") {
		parts := strings.SplitN(target, "@", 2)
		pkgName = parts[0]
		targetVer = parts[1]
	} else {
		pkgName = target
	}

	// Load Config
	cfg, err := config.LoadDefault()
	if err != nil {
		output.Warning("Failed to load config, using defaults: %v", err)
		cfg = config.Default()
	}

	// Initialize Cache
	c := cache.NewFileSystemCache()

	// Cache locations
	// TODO: Load from config
	cacheDirs := []string{
		"/var/cache/pacman/pkg",
		c.GetDir(),
	}

	// Find versions in all locations
	var candidates []cache.CachedPackage

	for _, dir := range cacheDirs {
		if matches, err := c.FindVersions(dir, pkgName); err == nil {
			candidates = append(candidates, matches...)
		}
	}

	if len(candidates) == 0 {
		return fmt.Errorf("no cached versions found for package '%s'", pkgName)
	}

	// Sort candidates by version (Newest first) using Alpm VerCmp
	sort.Slice(candidates, func(i, j int) bool {
		// Descending order: v1 > v2 returned true
		return alpm.VerCmp(candidates[i].Version, candidates[j].Version) > 0
	})

	if rollbackList {
		output.Info("Available versions for %s:", pkgName)
		for _, p := range candidates {
			fmt.Printf("  %s (%s)\n", p.Version, p.Path)
		}
		return nil
	}

	// Determine file to install
	var fileToInstall string

	if targetVer != "" {
		for _, p := range candidates {
			if p.Version == targetVer || strings.Contains(p.Version, targetVer) {
				fileToInstall = p.Path
				break
			}
		}
		if fileToInstall == "" {
			return fmt.Errorf("version '%s' not found in cache", targetVer)
		}
	} else {
		// Auto-rollback strategy
		eng, err := shedman.NewEngineWithConfig(cfg)
		if err != nil {
			return err
		}

		info, err := eng.Info(pkgName)
		if err == nil {
			// Find first candidate strictly OLDER than current version
			// OR just different if downgrade isn't strictly enforced (e.g. reinstalling same version)
			// Generally rollback implies downgrade.
			for _, p := range candidates {
				// if candidate < current
				if alpm.VerCmp(p.Version, info.Version) < 0 {
					fileToInstall = p.Path
					break
				}
			}

			// If no older version found, maybe we are on newest and want the previous one even if it's same?
			// Fallback: take 2nd candidate if 1st is current
			if fileToInstall == "" && len(candidates) > 1 {
				if candidates[0].Version == info.Version {
					fileToInstall = candidates[1].Path
				}
			}
		}

		if fileToInstall == "" {
			// Fallback: just take the latest available if we couldn't determine installed version
			// OR if installed version > all cached versions, take the newest cached one.
			if len(candidates) > 0 {
				fileToInstall = candidates[0].Path
			}
		}
	}

	output.Info("Rolling back %s to %s", pkgName, fileToInstall)

	if !rollbackYes {
		if !output.Confirm("Proceed with rollback?", output.ConfirmOptions{Default: false}) {
			return fmt.Errorf("rollback cancelled")
		}
	}

	// Execute Install
	eng, err := shedman.NewEngineWithConfig(cfg)
	if err != nil {
		return err
	}

	opts := backend.InstallOptions{
		NoConfirm: rollbackYes,
	}

	return eng.InstallFile(fileToInstall, opts)
}

func init() {
	rootCmd.AddCommand(NewRollbackCmd())
}
