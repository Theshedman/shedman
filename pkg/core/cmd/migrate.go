package cmd

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/alpm"
	"github.com/theshedman/shedman/internal/config"
)

var (
	migrateFrom   string
	migrateDryRun bool
)

// MigrateCmd imports pacman.conf settings into shedman config.
var MigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Import pacman configuration into shedman",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := migrateFrom
		if path == "" {
			path = alpm.DefaultPacmanConfPath
		}

		cfg, err := config.LoadDefault()
		if err != nil {
			cfg = config.Default()
		}

		return RunMigrate(cmd.OutOrStdout(), cfg, path, migrateDryRun)
	},
}

func init() {
	MigrateCmd.Flags().StringVar(&migrateFrom, "from", "", "Path to pacman.conf")
	MigrateCmd.Flags().BoolVar(&migrateDryRun, "dry-run", false, "Show what would be imported without writing config")
}

type migrateChange struct {
	Field string
	Old   string
	New   string
}

func RunMigrate(w io.Writer, cfg *config.Config, pacmanPath string, dryRun bool) error {
	conf, err := alpm.ParsePacmanConf(pacmanPath)
	if err != nil {
		return fmt.Errorf("failed to parse pacman.conf: %w", err)
	}

	changes := make([]migrateChange, 0)

	oldIgnore := strings.Join(cfg.Packages.IgnorePkg, ",")
	cfg.Packages.IgnorePkg = conf.IgnorePkg
	addChange(&changes, "packages.ignore_pkg", oldIgnore, strings.Join(cfg.Packages.IgnorePkg, ","))

	oldIgnoreGroup := strings.Join(cfg.Packages.IgnoreGroup, ",")
	cfg.Packages.IgnoreGroup = conf.IgnoreGroup
	addChange(&changes, "packages.ignore_group", oldIgnoreGroup, strings.Join(cfg.Packages.IgnoreGroup, ","))

	oldHold := strings.Join(cfg.Packages.HoldPkg, ",")
	cfg.Packages.HoldPkg = conf.HoldPkg
	addChange(&changes, "packages.hold_pkg", oldHold, strings.Join(cfg.Packages.HoldPkg, ","))

	oldGPG := cfg.Security.GPGDir
	cfg.Security.GPGDir = conf.GPGDir
	addChange(&changes, "security.gpg_dir", oldGPG, cfg.Security.GPGDir)

	oldSig := cfg.Security.SigLevel
	cfg.Security.SigLevel = conf.SigLevel
	addChange(&changes, "security.sig_level", oldSig, cfg.Security.SigLevel)

	oldLog := cfg.Logging.File
	cfg.Logging.File = conf.LogFile
	addChange(&changes, "logging.file", oldLog, cfg.Logging.File)

	if conf.ParallelDownloads > 0 {
		oldParallel := fmt.Sprintf("%d", cfg.Network.ParallelDownloads)
		cfg.Network.ParallelDownloads = conf.ParallelDownloads
		addChange(&changes, "network.parallel_downloads", oldParallel, fmt.Sprintf("%d", cfg.Network.ParallelDownloads))
	}

	archMirrors := collectArchMirrors(conf)
	if len(archMirrors) > 0 {
		oldArch := strings.Join(cfg.Mirrors.Arch, ",")
		cfg.Mirrors.Arch = archMirrors
		addChange(&changes, "mirrors.arch", oldArch, strings.Join(cfg.Mirrors.Arch, ","))
	}

	if len(changes) == 0 {
		_, _ = fmt.Fprintln(w, "No changes detected.")
		return nil
	}

	_, _ = fmt.Fprintf(w, "Importing pacman config from %s:\n", pacmanPath)
	for _, change := range changes {
		_, _ = fmt.Fprintf(w, "  - %s: %s -> %s\n", change.Field, change.Old, change.New)
	}

	if dryRun {
		_, _ = fmt.Fprintln(w, "Dry-run: no changes written.")
		return nil
	}

	if err := config.Save(config.DefaultConfigPath(), cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	_, _ = fmt.Fprintln(w, "Migration complete.")
	return nil
}

func addChange(changes *[]migrateChange, field, oldVal, newVal string) {
	if oldVal == newVal {
		return
	}
	*changes = append(*changes, migrateChange{
		Field: field,
		Old:   oldVal,
		New:   newVal,
	})
}

func collectArchMirrors(conf *alpm.PacmanConf) []string {
	if conf == nil {
		return nil
	}

	mirrorSet := make(map[string]bool)
	for _, repo := range conf.Repositories {
		for _, server := range repo.Servers {
			expanded := conf.ExpandVariables(server, repo.Name)
			if expanded == "" {
				continue
			}
			mirrorSet[expanded] = true
		}
	}

	if len(mirrorSet) == 0 {
		return nil
	}

	var mirrors []string
	for m := range mirrorSet {
		mirrors = append(mirrors, m)
	}
	sort.Strings(mirrors)
	return mirrors
}
