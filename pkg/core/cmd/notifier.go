package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/internal/output"
	"github.com/theshedman/shedman/pkg/executor"
	"github.com/theshedman/shedman/pkg/notifier"
)

var notifierQuiet bool

// NotifierCmd manages update notifications.
var NotifierCmd = &cobra.Command{
	Use:   "notifier",
	Short: "Manage update notifications",
	Long:  "Enable, disable, and manually check for update notifications.",
}

var notifierEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable update notifications",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefault()
		if err != nil {
			return err
		}
		return enableNotifierTimer(cfg)
	},
}

var notifierDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable update notifications",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSystemctlUser("disable", "--now", "shedman-notifier.timer")
	},
}

var notifierStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show notifier status",
	RunE: func(cmd *cobra.Command, args []string) error {
		enabled, err := isSystemctlUserEnabled("shedman-notifier.timer")
		if err != nil {
			return err
		}
		state := "disabled"
		if enabled {
			state = "enabled"
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Notifier status: %s\n", state)
		return nil
	},
}

var notifierCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check for updates and notify",
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}
		cfg := engine.GetConfig()
		if cfg == nil {
			cfg = config.Default()
		}

		diffs, err := engine.Diff()
		if err != nil {
			return err
		}

		if len(diffs) == 0 {
			_ = writeDefaultUpdateCount(0)
			if !notifierQuiet {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No updates available.")
			}
			return nil
		}

		if cfg.Notifications.Desktop {
			n := notifier.New()
			if n != nil {
				msg := fmt.Sprintf("%d packages can be updated.\nRun: shedman update", len(diffs))
				_ = n.Notify("ShedOS Updates Available", msg, "info")
			}
		}

		_ = writeDefaultUpdateCount(len(diffs))
		if !notifierQuiet {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%d updates available.\n", len(diffs))
		}
		return nil
	},
}

func init() {
	NotifierCmd.AddCommand(notifierEnableCmd)
	NotifierCmd.AddCommand(notifierDisableCmd)
	NotifierCmd.AddCommand(notifierCheckCmd)
	NotifierCmd.AddCommand(notifierStatusCmd)

	notifierCheckCmd.Flags().BoolVar(&notifierQuiet, "quiet", false, "Suppress output (for timers)")
}

func enableNotifierTimer(cfg *config.Config) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	unitDir := filepath.Join(home, ".config", "systemd", "user")
	if err := os.MkdirAll(unitDir, 0755); err != nil {
		return err
	}

	interval := cfg.Notifications.GetInterval()
	intervalStr := interval.String()
	if interval <= 0 {
		intervalStr = "6h"
	}

	servicePath := filepath.Join(unitDir, "shedman-notifier.service")
	timerPath := filepath.Join(unitDir, "shedman-notifier.timer")

	service := strings.Join([]string{
		"[Unit]",
		"Description=Shedman update notifier",
		"",
		"[Service]",
		"Type=oneshot",
		"ExecStart=shedman notifier check --quiet",
		"",
	}, "\n")

	timer := strings.Join([]string{
		"[Unit]",
		"Description=Shedman update notifier timer",
		"",
		"[Timer]",
		"OnBootSec=5min",
		fmt.Sprintf("OnUnitActiveSec=%s", intervalStr),
		"Persistent=true",
		"",
		"[Install]",
		"WantedBy=timers.target",
		"",
	}, "\n")

	if err := os.WriteFile(servicePath, []byte(service), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(timerPath, []byte(timer), 0644); err != nil {
		return err
	}

	if err := ensureMotdScript(); err != nil {
		output.Warning("Failed to install MOTD script: %v", err)
	}

	if err := runSystemctlUser("daemon-reload"); err != nil {
		return err
	}
	return runSystemctlUser("enable", "--now", "shedman-notifier.timer")
}

func runSystemctlUser(args ...string) error {
	exec := &executor.RealExecutor{}
	cmd := exec.Command("systemctl", append([]string{"--user"}, args...)...)
	return cmd.Run()
}

func isSystemctlUserEnabled(unit string) (bool, error) {
	exec := &executor.RealExecutor{}
	cmd := exec.Command("systemctl", "--user", "is-enabled", unit)
	out, err := cmd.Output()
	if err != nil {
		return false, nil
	}
	return strings.TrimSpace(string(out)) == "enabled", nil
}

func updateCountPath(home string) string {
	return filepath.Join(home, ".cache", "shedman", "update-count")
}

func writeDefaultUpdateCount(count int) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	return writeUpdateCount(updateCountPath(home), count)
}

func writeUpdateCount(path string, count int) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data := []byte(fmt.Sprintf("%d\n", count))
	return os.WriteFile(path, data, 0644)
}

func motdScriptContent() string {
	return strings.Join([]string{
		"#!/bin/sh",
		"COUNT_FILE=\"$HOME/.cache/shedman/update-count\"",
		"if [ -f \"$COUNT_FILE\" ]; then",
		"  COUNT=$(cat \"$COUNT_FILE\")",
		"  if [ \"$COUNT\" -gt 0 ] 2>/dev/null; then",
		"    echo \"🔔 $COUNT updates available. Run: shedman update\"",
		"  fi",
		"fi",
		"",
	}, "\n")
}

func ensureMotdScript() error {
	path := "/etc/profile.d/shedman-motd.sh"
	content := motdScriptContent()
	return os.WriteFile(path, []byte(content), 0755)
}
