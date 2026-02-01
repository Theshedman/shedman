package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/internal/output"
	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/core/providers/aur"
	shedrepo "github.com/theshedman/shedman/pkg/core/providers/shed"
	"github.com/theshedman/shedman/pkg/theme"
)

var (
	themeInstalledOnly bool
	themeRollbackList  bool
	themePreview       bool
)

// ThemeCmd represents theme management.
var ThemeCmd = &cobra.Command{
	Use:   "theme",
	Short: "Manage system themes",
	Long:  "List, install, apply, and rollback system themes.",
}

var themeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available themes",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefault()
		if err != nil {
			output.Warning("Failed to load config, using defaults: %v", err)
			cfg = config.Default()
		}

		engine := core.NewEngine()
		officialBackend, err := DetectBackendWithConfig(&cfg.Backend)
		if err == nil {
			engine.SetOfficialBackend(officialBackend)
		}

		if cfg.AUR.Enabled && core.IsArchBased() {
			pkgCache := core.NewPackageFileCacheWithBackend(24*time.Hour, officialBackend)
			engine.AddBackend(aur.NewWithURL(cfg.Mirrors.AUR, pkgCache))
		}

		fsCache := core.NewFileSystemCache()
		timeout := 30 * time.Second
		if cfg.Network.Timeout > 0 {
			timeout = time.Duration(cfg.Network.Timeout) * time.Second
		}
		if len(cfg.Mirrors.ShedOS) > 0 {
			engine.AddBackend(shedrepo.NewWithMirrors(cfg.Mirrors.ShedOS, fsCache, timeout))
		} else {
			engine.AddBackend(shedrepo.New(fsCache, timeout))
		}

		mgr := theme.New(engine)
		themes, err := mgr.List()
		if err != nil {
			return err
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
		_, _ = fmt.Fprintln(w, "NAME\tVERSION\tINSTALLED\tDESCRIPTION")
		for _, t := range themes {
			installed := ""
			if engine.IsInstalled(t.Name) {
				installed = "yes"
			}
			if themeInstalledOnly && installed == "" {
				continue
			}
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", t.Name, t.Version, installed, t.Description)
		}
		_ = w.Flush()
		return nil
	},
}

var themeApplyCmd = &cobra.Command{
	Use:   "apply <theme>",
	Short: "Apply a theme (install package)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return applyTheme(cmd.OutOrStdout(), args[0], themePreview)
	},
}

var themeInstallCmd = &cobra.Command{
	Use:   "install <theme>",
	Short: "Install a theme",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return applyTheme(cmd.OutOrStdout(), args[0], themePreview)
	},
}

var themeRollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback to the previously applied theme",
	RunE: func(cmd *cobra.Command, args []string) error {
		state, err := loadThemeState()
		if err != nil {
			return err
		}

		if themeRollbackList {
			if len(state.History) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No theme history found.")
				return nil
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
			_, _ = fmt.Fprintln(w, "INDEX\tTHEME\tCURRENT")
			for i, name := range state.History {
				current := ""
				if i == len(state.History)-1 {
					current = "yes"
				}
				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\n", i+1, name, current)
			}
			_ = w.Flush()
			return nil
		}

		if len(state.History) < 2 {
			return fmt.Errorf("no previous theme to rollback to")
		}

		target := state.History[len(state.History)-2]
		if err := applyTheme(cmd.OutOrStdout(), target, false); err != nil {
			return err
		}

		state.History = state.History[:len(state.History)-1]
		if err := saveThemeState(state); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Rolled back to %s.\n", target)
		return nil
	},
}

func init() {
	ThemeCmd.AddCommand(themeListCmd)
	ThemeCmd.AddCommand(themeInstallCmd)
	ThemeCmd.AddCommand(themeApplyCmd)
	ThemeCmd.AddCommand(themeRollbackCmd)

	themeListCmd.Flags().BoolVar(&themeInstalledOnly, "installed", false, "Show installed themes only")
	themeRollbackCmd.Flags().BoolVar(&themeRollbackList, "list", false, "List theme history")
	themeApplyCmd.Flags().BoolVar(&themePreview, "preview", false, "Preview theme before applying")
}

func applyTheme(w io.Writer, name string, preview bool) error {
	engine, err := NewEngineWithConfig(nil)
	if err != nil {
		return err
	}
	if preview {
		if err := previewTheme(w, engine, name); err != nil {
			return err
		}
	}
	mgr := theme.New(engine)
	if err := mgr.Apply(name); err != nil {
		return err
	}
	if err := updateThemeState(name); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(w, "Theme applied.")
	return nil
}

func previewTheme(w io.Writer, engine *core.Engine, name string) error {
	if engine == nil {
		return fmt.Errorf("engine not available")
	}

	pkgName := name
	if !strings.HasPrefix(name, "shedos-theme-") {
		pkgName = "shedos-theme-" + name
	}

	info, err := engine.Info(pkgName)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(w, "Theme: %s\n", name)
	_, _ = fmt.Fprintf(w, "Version: %s\n", info.Version)
	if info.Description != "" {
		_, _ = fmt.Fprintf(w, "Description: %s\n", info.Description)
	}
	return nil
}

type themeState struct {
	History []string `json:"history"`
}

func updateThemeState(name string) error {
	state, err := loadThemeState()
	if err != nil {
		return err
	}
	if len(state.History) > 0 && state.History[len(state.History)-1] == name {
		return nil
	}
	state.History = append(state.History, name)
	return saveThemeState(state)
}

func themeStatePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state", "shedman", "theme.json"), nil
}

func loadThemeState() (*themeState, error) {
	path, err := themeStatePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &themeState{}, nil
		}
		return nil, err
	}

	var state themeState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func saveThemeState(state *themeState) error {
	path, err := themeStatePath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
