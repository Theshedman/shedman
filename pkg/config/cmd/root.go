package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	appconfig "github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/internal/output"
	"github.com/theshedman/shedman/internal/util"
	pkgconfig "github.com/theshedman/shedman/pkg/config"
	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/core/providers/pacman"
	"github.com/theshedman/shedman/pkg/tui"
)

// ConfigCmd is the root command for configuration management
var ConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration packages",
	Long:  "List, diff, apply, reset, and rollback configuration packages managed by shedman.",
}

func init() {
	ConfigCmd.AddCommand(newListCmd())
	ConfigCmd.AddCommand(newDiffCmd())
	ConfigCmd.AddCommand(newApplyCmd())
	ConfigCmd.AddCommand(newResetCmd())
	ConfigCmd.AddCommand(newRollbackCmd())
	ConfigCmd.AddCommand(newStatusCmd())
	ConfigCmd.AddCommand(newConfigGetCmd())
	ConfigCmd.AddCommand(newConfigSetCmd())
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available configuration packages",
		RunE: func(cmd *cobra.Command, args []string) error {
			eng, _, err := newCoreEngine()
			if err != nil {
				return err
			}

			mgr := pkgconfig.New(eng)
			configs, err := mgr.List()
			if err != nil {
				return err
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
			_, _ = fmt.Fprintln(w, "NAME\tVERSION\tINSTALLED\tDESCRIPTION")

			backend := eng.GetOfficialBackend()
			for _, c := range configs {
				installed := ""
				if backend != nil && backend.IsInstalled(c.Name) {
					installed = "yes"
				}
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", c.Name, c.Version, installed, c.Description)
			}
			_ = w.Flush()
			return nil
		},
	}
}

func newDiffCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diff [config]",
		Short: "Show differences between installed configs and defaults",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			eng, _, err := newCoreEngine()
			if err != nil {
				return err
			}

			cfgEngine := createConfigEngine(eng, false, tui.NewConflictResolver())
			packages, err := resolveConfigPackages(eng, args)
			if err != nil {
				return err
			}

			home, _ := os.UserHomeDir()
			anyDiff := false

			for _, pkgName := range packages {
				found, err := diffConfigPackage(cmd.OutOrStdout(), eng, cfgEngine, pkgName, home)
				if err != nil {
					return err
				}
				if found {
					anyDiff = true
				}
			}

			if !anyDiff {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No differences found.")
			}
			return nil
		},
	}
}

var (
	applyYes    bool
	applyBackup bool
	applyForce  bool
	applyMerge  bool
	resetBackup bool
)

func newApplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply [config]",
		Short: "Apply default configuration files",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			eng, cfg, err := newCoreEngine()
			if err != nil {
				return err
			}

			packages, err := resolveConfigPackages(eng, args)
			if err != nil {
				return err
			}

			resolver, mergeResolver := buildConflictResolver(cfg)
			cfgEngine := createConfigEngine(eng, applyBackup, resolver)
			if mergeResolver != nil {
				mergeResolver.engine = cfgEngine
			}

			home, _ := os.UserHomeDir()

			for _, pkgName := range packages {
				if err := ensureConfigPackageInstalled(eng, cfg, pkgName, applyYes || applyForce); err != nil {
					return err
				}

				if err := applyConfigPackage(cmd.OutOrStdout(), eng, cfgEngine, pkgName, home, mergeResolver); err != nil {
					return err
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&applyYes, "yes", "y", false, "Skip per-file prompts (apply defaults)")
	cmd.Flags().BoolVar(&applyBackup, "backup", true, "Backup existing files before overwriting")
	cmd.Flags().BoolVar(&applyForce, "force", false, "Overwrite without prompting")
	cmd.Flags().BoolVar(&applyMerge, "merge", false, "Open merge tool for conflicts")
	return cmd
}

func newResetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset <config>",
		Short: "Reset configuration files to defaults",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			eng, _, err := newCoreEngine()
			if err != nil {
				return err
			}

			cfgEngine := createConfigEngine(eng, resetBackup, staticResolver{action: pkgconfig.ActionUpdate})
			pkgName := normalizeConfigName(args[0])

			home, _ := os.UserHomeDir()
			return resetConfigPackage(cmd.OutOrStdout(), eng, cfgEngine, pkgName, home)
		},
	}
	cmd.Flags().BoolVar(&resetBackup, "backup", true, "Backup existing files before overwriting")
	return cmd
}

var rollbackList bool

func newRollbackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rollback <config> [timestamp]",
		Short: "Rollback configuration files to previous backups",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			pkgName := normalizeConfigName(args[0])
			var timestamp string
			if len(args) > 1 {
				timestamp = args[1]
			}

			eng, _, err := newCoreEngine()
			if err != nil {
				return err
			}

			home, _ := os.UserHomeDir()

			if rollbackList {
				return listConfigBackups(cmd.OutOrStdout(), eng, pkgName, home)
			}

			return rollbackConfigPackage(cmd.OutOrStdout(), eng, pkgName, home, timestamp)
		},
	}

	cmd.Flags().BoolVar(&rollbackList, "list", false, "List available backups")
	return cmd
}

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show status of tracked configuration files",
		Run: func(cmd *cobra.Command, args []string) {
			// Use default state path
			home, _ := os.UserHomeDir()
			statePath := filepath.Join(home, ".local", "state", "shedman", "configs.json")

			stateMgr := pkgconfig.NewJSONStateManager(statePath)
			if err := stateMgr.Load(); err != nil {
				_, _ = fmt.Printf("Failed to load state: %v\n", err)

				return
			}

			states := stateMgr.List()

			if len(states) == 0 {
				output.Info("No configuration files are currently tracked.")
				return
			}

			cmd.Printf("%-50s %-20s %s\n", "PATH", "VERSION", "LAST MODIFIED")
			cmd.Printf("%-50s %-20s %s\n", "----", "-------", "-------------")
			for _, s := range states {
				cmd.Printf("%-50s %-20s %s\n", s.Path, s.Version, s.LastModified.Format("2006-01-02 15:04:05"))
			}
		},
	}
	cmd.Hidden = true
	return cmd
}

type staticResolver struct {
	action pkgconfig.Action
}

func (r staticResolver) Resolve(_ string, _ string) (pkgconfig.Action, error) {
	return r.action, nil
}

type mergeResolver struct {
	engine    *pkgconfig.ConfigEngine
	editor    string
	tempFiles map[string]string
	packages  map[string]string
}

func (r *mergeResolver) Resolve(file string, _ string) (pkgconfig.Action, error) {
	tmpPath, ok := r.tempFiles[file]
	if !ok {
		return pkgconfig.ActionKeepUser, nil
	}

	if err := runMergeTool(r.editor, file, tmpPath); err != nil {
		return pkgconfig.ActionKeepUser, err
	}

	if r.engine != nil {
		hash, err := r.engine.Differ.CalculateHash(file)
		if err == nil {
			pkgName := r.packages[file]
			r.engine.StateMgr.Set(pkgName, file, pkgconfig.FileState{
				Path:         file,
				Hash:         hash,
				LastModified: time.Now(),
				Version:      "merged",
			})
			_ = r.engine.StateMgr.Save()
		}
	}

	return pkgconfig.ActionKeepUser, nil
}

func buildConflictResolver(cfg *appconfig.Config) (pkgconfig.ConflictResolver, *mergeResolver) {
	if applyForce || applyYes {
		return staticResolver{action: pkgconfig.ActionUpdate}, nil
	}

	if applyMerge {
		resolver := &mergeResolver{
			editor:    resolveEditor(cfg),
			tempFiles: make(map[string]string),
			packages:  make(map[string]string),
		}
		return resolver, resolver
	}

	return tui.NewConflictResolver(), nil
}

func resolveEditor(cfg *appconfig.Config) string {
	if cfg != nil && cfg.General.Editor != "" {
		return cfg.General.Editor
	}
	if env := os.Getenv("EDITOR"); env != "" {
		return env
	}
	return "vimdiff"
}

// mergeToolValidator validates the merge tool path. Can be replaced in tests.
var mergeToolValidator = util.ValidateExecutablePath

func runMergeTool(editor, target, source string) error {
	// Default to vimdiff if no editor specified
	if editor == "" {
		editor = "vimdiff"
	}

	// Validate the merge tool path to prevent command injection
	// We only accept the command name, not arguments embedded in the string
	if err := mergeToolValidator(editor); err != nil {
		return fmt.Errorf("invalid merge tool: %w", err)
	}

	proc := exec.Command(editor, target, source)
	proc.Stdin = os.Stdin
	proc.Stdout = os.Stdout
	proc.Stderr = os.Stderr
	return proc.Run()
}

func newCoreEngine() (*core.Engine, *appconfig.Config, error) {
	cfg, err := appconfig.LoadDefault()
	if err != nil {
		output.Warning("Failed to load config, using defaults: %v", err)
		cfg = appconfig.Default()
	}

	backend, err := pacman.NewWithConfig(pacman.DefaultConfig())
	if err != nil {
		return nil, cfg, fmt.Errorf("failed to initialize backend: %w", err)
	}

	eng := core.NewEngineWithBackend(backend)
	eng.SetConfig(cfg)
	return eng, cfg, nil
}

func createConfigEngine(eng *core.Engine, backup bool, resolver pkgconfig.ConflictResolver) *pkgconfig.ConfigEngine {
	home, _ := os.UserHomeDir()
	statePath := filepath.Join(home, ".local", "state", "shedman", "configs.json")

	stateMgr := pkgconfig.NewJSONStateManager(statePath)
	_ = stateMgr.Load()

	var backupMgr pkgconfig.BackupManager
	if backup {
		backupMgr = pkgconfig.NewFileBackupManager()
	} else {
		backupMgr = noopBackupManager{}
	}

	differ := pkgconfig.NewDiffer()
	provider := pkgconfig.NewPacmanSourceProvider(eng)

	if resolver == nil {
		resolver = tui.NewConflictResolver()
	}

	return pkgconfig.NewConfigEngine(stateMgr, backupMgr, differ, resolver, provider)
}

type noopBackupManager struct{}

func (n noopBackupManager) Backup(string) (string, error) { return "", nil }
func (n noopBackupManager) Rotate(string, int) error      { return nil }

func resolveConfigPackages(eng *core.Engine, args []string) ([]string, error) {
	if len(args) > 0 {
		name := normalizeConfigName(args[0])
		return []string{name}, nil
	}
	return listInstalledConfigPackages(eng)
}

func listInstalledConfigPackages(eng *core.Engine) ([]string, error) {
	backend := eng.GetOfficialBackend()
	if backend == nil {
		return nil, core.ErrBackendNotFound
	}

	pkgs, err := backend.GetInstalledPackages()
	if err != nil {
		return nil, err
	}

	var configs []string
	for _, p := range pkgs {
		if strings.HasPrefix(p.Name, "shedos-configs-") {
			configs = append(configs, p.Name)
		}
	}
	sort.Strings(configs)

	if len(configs) == 0 {
		return nil, fmt.Errorf("no configuration packages are installed")
	}

	return configs, nil
}

func normalizeConfigName(name string) string {
	if strings.HasPrefix(name, "shedos-configs-") {
		return name
	}
	return "shedos-configs-" + name
}

func ensureConfigPackageInstalled(eng *core.Engine, cfg *appconfig.Config, pkgName string, noConfirm bool) error {
	backend := eng.GetOfficialBackend()
	if backend != nil && backend.IsInstalled(pkgName) {
		return nil
	}

	opts := core.InstallOptions{
		Needed:    true,
		NoConfirm: noConfirm || (cfg != nil && !cfg.General.Confirm),
	}
	return eng.Install([]string{pkgName}, opts)
}

type configTarget struct {
	Package     string
	PackagePath string
	TargetPath  string
}

func configTargets(eng *core.Engine, pkgName, home string) ([]configTarget, error) {
	backend := eng.GetOfficialBackend()
	if backend == nil {
		return nil, core.ErrBackendNotFound
	}

	files, err := backend.GetPackageFiles(pkgName)
	if err != nil {
		return nil, err
	}

	var targets []configTarget
	for _, file := range files {
		if strings.HasSuffix(file, "/") {
			continue
		}
		targetPath := mapConfigTarget(file, home)
		targets = append(targets, configTarget{
			Package:     pkgName,
			PackagePath: file,
			TargetPath:  targetPath,
		})
	}

	return targets, nil
}

func mapConfigTarget(path, home string) string {
	const skelPrefix = "/etc/skel/"
	if strings.HasPrefix(path, skelPrefix) {
		return filepath.Join(home, strings.TrimPrefix(path, skelPrefix))
	}
	return path
}

func diffConfigPackage(w io.Writer, eng *core.Engine, cfgEngine *pkgconfig.ConfigEngine, pkgName, home string) (bool, error) {
	targets, err := configTargets(eng, pkgName, home)
	if err != nil {
		return false, err
	}

	found := false
	headerPrinted := false

	for _, target := range targets {
		original, err := cfgEngine.GetOriginal(target.PackagePath)
		if err != nil {
			output.Warning("Skipping %s: %v", target.PackagePath, err)
			continue
		}

		current, err := os.ReadFile(target.TargetPath)
		if err != nil {
			if !os.IsNotExist(err) {
				output.Warning("Skipping %s: %v", target.TargetPath, err)
				continue
			}
			current = []byte{}
		}

		diff, err := cfgEngine.Differ.GenerateDiff(target.TargetPath, string(current), "package-default", string(original))
		if err != nil {
			return false, err
		}
		if diff == "" {
			continue
		}

		if !headerPrinted {
			_, _ = fmt.Fprintf(w, "Config package: %s\n", pkgName)
			headerPrinted = true
		}

		_, _ = fmt.Fprintln(w, diff)
		found = true
	}

	return found, nil
}

func applyConfigPackage(w io.Writer, eng *core.Engine, cfgEngine *pkgconfig.ConfigEngine, pkgName, home string, merge *mergeResolver) error {
	targets, err := configTargets(eng, pkgName, home)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(w, "Applying %s...\n", pkgName)

	for _, target := range targets {
		original, err := cfgEngine.GetOriginal(target.PackagePath)
		if err != nil {
			output.Warning("Skipping %s: %v", target.PackagePath, err)
			continue
		}

		tmpFile, err := os.CreateTemp("", "shedman-config-*.conf")
		if err != nil {
			return err
		}
		if _, err := tmpFile.Write(original); err != nil {
			_ = tmpFile.Close()
			_ = os.Remove(tmpFile.Name())
			return err
		}
		_ = tmpFile.Close()

		if merge != nil {
			merge.tempFiles[target.TargetPath] = tmpFile.Name()
			merge.packages[target.TargetPath] = pkgName
		}

		if err := cfgEngine.Apply(pkgName, tmpFile.Name(), target.TargetPath); err != nil {
			_ = os.Remove(tmpFile.Name())
			return err
		}

		if merge != nil {
			delete(merge.tempFiles, target.TargetPath)
			delete(merge.packages, target.TargetPath)
		}

		_ = os.Remove(tmpFile.Name())
	}

	_, _ = fmt.Fprintf(w, "Applied %s.\n", pkgName)
	return nil
}

func resetConfigPackage(w io.Writer, eng *core.Engine, cfgEngine *pkgconfig.ConfigEngine, pkgName, home string) error {
	targets, err := configTargets(eng, pkgName, home)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(w, "Resetting %s...\n", pkgName)

	for _, target := range targets {
		original, err := cfgEngine.GetOriginal(target.PackagePath)
		if err != nil {
			output.Warning("Skipping %s: %v", target.PackagePath, err)
			continue
		}

		tmpFile, err := os.CreateTemp("", "shedman-config-reset-*.conf")
		if err != nil {
			return err
		}
		if _, err := tmpFile.Write(original); err != nil {
			_ = tmpFile.Close()
			_ = os.Remove(tmpFile.Name())
			return err
		}
		_ = tmpFile.Close()

		if err := cfgEngine.Reset(pkgName, tmpFile.Name(), target.TargetPath); err != nil {
			_ = os.Remove(tmpFile.Name())
			return err
		}
		_ = os.Remove(tmpFile.Name())
	}

	_, _ = fmt.Fprintf(w, "Reset %s.\n", pkgName)
	return nil
}

type backupInfo struct {
	Path      string
	Timestamp string
	Parsed    time.Time
}

func listConfigBackups(w io.Writer, eng *core.Engine, pkgName, home string) error {
	targets, err := configTargets(eng, pkgName, home)
	if err != nil {
		return err
	}

	var backups []backupInfo
	for _, target := range targets {
		list, err := listBackups(target.TargetPath)
		if err != nil {
			return err
		}
		backups = append(backups, list...)
	}

	if len(backups) == 0 {
		_, _ = fmt.Fprintln(w, "No backups found.")
		return nil
	}

	sort.Slice(backups, func(i, j int) bool {
		if backups[i].Parsed.Equal(backups[j].Parsed) {
			return backups[i].Path < backups[j].Path
		}
		return backups[i].Parsed.After(backups[j].Parsed)
	})

	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	_, _ = fmt.Fprintln(tw, "TIMESTAMP\tFILE")
	for _, b := range backups {
		_, _ = fmt.Fprintf(tw, "%s\t%s\n", b.Timestamp, b.Path)
	}
	_ = tw.Flush()

	return nil
}

func rollbackConfigPackage(w io.Writer, eng *core.Engine, pkgName, home, timestamp string) error {
	targets, err := configTargets(eng, pkgName, home)
	if err != nil {
		return err
	}

	restored := 0
	for _, target := range targets {
		backup, err := selectBackup(target.TargetPath, timestamp)
		if err != nil {
			return err
		}
		if backup == "" {
			continue
		}

		if err := restoreBackup(target.TargetPath, backup); err != nil {
			return err
		}
		restored++
	}

	if restored == 0 {
		_, _ = fmt.Fprintln(w, "No backups restored.")
		return nil
	}

	_, _ = fmt.Fprintf(w, "Restored %d file(s).\n", restored)
	return nil
}

func listBackups(path string) ([]backupInfo, error) {
	pattern := path + ".*.bak"
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	var backups []backupInfo
	for _, match := range matches {
		ts := strings.TrimSuffix(match, ".bak")
		ts = strings.TrimPrefix(ts, path+".")
		parsed, _ := time.Parse("20060102150405.000", ts)
		backups = append(backups, backupInfo{
			Path:      match,
			Timestamp: ts,
			Parsed:    parsed,
		})
	}
	return backups, nil
}

func selectBackup(path, timestamp string) (string, error) {
	backups, err := listBackups(path)
	if err != nil {
		return "", err
	}
	if len(backups) == 0 {
		return "", nil
	}

	if timestamp == "" {
		sort.Slice(backups, func(i, j int) bool {
			return backups[i].Parsed.After(backups[j].Parsed)
		})
		return backups[0].Path, nil
	}

	for _, b := range backups {
		if b.Timestamp == timestamp {
			return b.Path, nil
		}
	}
	return "", nil
}

func restoreBackup(target, backupPath string) error {
	input, err := os.ReadFile(backupPath)
	if err != nil {
		return err
	}
	return os.WriteFile(target, input, 0600)
}
