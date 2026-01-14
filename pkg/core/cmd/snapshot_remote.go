package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/snapshot"
)

var SnapshotRemoteCmd = &cobra.Command{
	Use:   "remote",
	Short: "Manage remote snapshots",
}

var SnapshotRemotePushCmd = &cobra.Command{
	Use:   "push <id> <target-name> [path]",
	Short: "Push a snapshot to a remote target",
	Args:  cobra.RangeArgs(2, 3), // id, target, optional path
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}

		path := ""
		if len(args) > 2 {
			path = args[2]
		}

		target := snapshot.RemoteTarget{
			Name: args[1],
			Path: path,
		}

		opts := snapshot.RemoteOptions{
			Compress:  snapshotRemoteCompress,
			Bandwidth: snapshotRemoteBandwidth,
			Delete:    snapshotRemoteDelete,
		}

		return RunSnapshotRemotePush(engine, args[0], target, opts, cmd.OutOrStdout())
	},
}

var SnapshotRemotePullCmd = &cobra.Command{
	Use:   "pull <id> <source-name> [path]",
	Short: "Pull a snapshot from a remote source",
	Args:  cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}

		path := ""
		if len(args) > 2 {
			path = args[2]
		}

		source := snapshot.RemoteTarget{
			Name: args[1],
			Path: path,
		}

		opts := snapshot.RemoteOptions{
			Compress:  snapshotRemoteCompress,
			Bandwidth: snapshotRemoteBandwidth,
			Delete:    snapshotRemoteDelete,
		}

		return RunSnapshotRemotePull(engine, args[0], source, opts, cmd.OutOrStdout())
	},
}

var SnapshotRemoteAddCmd = &cobra.Command{
	Use:   "add <name> <type> <path>",
	Short: "Add a new remote target",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}
		return RunSnapshotRemoteAdd(engine, args[0], args[1], args[2], cmd.OutOrStdout())
	},
}

var SnapshotRemoteListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured remotes",
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}
		return RunSnapshotRemoteList(engine, cmd.OutOrStdout())
	},
}

var SnapshotRemoteRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a configured remote",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}
		return RunSnapshotRemoteRemove(engine, args[0], cmd.OutOrStdout())
	},
}

var SnapshotRemoteTestCmd = &cobra.Command{
	Use:   "test <name>",
	Short: "Test connection to a remote",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}
		return RunSnapshotRemoteTest(engine, args[0], cmd.OutOrStdout())
	},
}

func init() {
	SnapshotRemoteCmd.AddCommand(SnapshotRemotePushCmd)
	SnapshotRemoteCmd.AddCommand(SnapshotRemotePullCmd)
	SnapshotRemoteCmd.AddCommand(SnapshotRemoteAddCmd)
	SnapshotRemoteCmd.AddCommand(SnapshotRemoteListCmd)
	SnapshotRemoteCmd.AddCommand(SnapshotRemoteRemoveCmd)
	SnapshotRemoteCmd.AddCommand(SnapshotRemoteTestCmd)

	// Flags for Push
	SnapshotRemotePushCmd.Flags().BoolVar(&snapshotRemoteCompress, "compress", true, "Enable compression")
	SnapshotRemotePushCmd.Flags().BoolVar(&snapshotRemoteDelete, "delete", false, "Delete extraneous files on destination")
	SnapshotRemotePushCmd.Flags().IntVar(&snapshotRemoteBandwidth, "bwlimit", 0, "Bandwidth limit in KB/s")

	// Flags for Pull
	SnapshotRemotePullCmd.Flags().BoolVar(&snapshotRemoteCompress, "compress", true, "Enable compression")
	SnapshotRemotePullCmd.Flags().BoolVar(&snapshotRemoteDelete, "delete", false, "Delete extraneous files on local (cautious usage)")
	SnapshotRemotePullCmd.Flags().IntVar(&snapshotRemoteBandwidth, "bwlimit", 0, "Bandwidth limit in KB/s")
}

var (
	snapshotRemoteCompress  bool
	snapshotRemoteDelete    bool
	snapshotRemoteBandwidth int
)

// Logic

func RunSnapshotRemotePush(engine *core.Engine, id string, target snapshot.RemoteTarget, opts snapshot.RemoteOptions, w io.Writer) error {
	mgr := engine.GetSnapshotManager()
	if mgr == nil {
		return fmt.Errorf("snapshot manager not available")
	}

	fmt.Fprintf(w, "Pushing snapshot %s to %s...\n", id, target.Name)
	if err := mgr.Push(id, target, opts); err != nil {
		return fmt.Errorf("push failed: %w", err)
	}
	fmt.Fprintln(w, "Push successful.")
	return nil
}

func RunSnapshotRemotePull(engine *core.Engine, id string, source snapshot.RemoteTarget, opts snapshot.RemoteOptions, w io.Writer) error {
	mgr := engine.GetSnapshotManager()
	if mgr == nil {
		return fmt.Errorf("snapshot manager not available")
	}

	fmt.Fprintf(w, "Pulling snapshot %s from %s...\n", id, source.Name)
	if err := mgr.Pull(id, source, opts); err != nil {
		return fmt.Errorf("pull failed: %w", err)
	}
	fmt.Fprintln(w, "Pull successful.")
	return nil
}

// --- Logic Implementations ---

func RunSnapshotRemoteAdd(engine *core.Engine, name, typ, path string, w io.Writer) error {
	// 1. Load Config (Mutable) - Engine has read-only. We need to load mutable.
	// For MVP, we might just modify the engine's config strictly in memory or fail if no write support?
	// But `config.Save` exists in many tools. We need a way to save.
	// Assume `config.LoadDefault` returns *Config and we can save it.

	// Re-load to ensure we have latest and file path
	cfg, err := LoadConfigForEdit() // Helper to be added or use config directly
	if err != nil {
		return fmt.Errorf("failed to load config for editing: %w", err)
	}

	if cfg.Snapshot.Remotes == nil {
		cfg.Snapshot.Remotes = make(map[string]config.RemoteConfig)
	}

	cfg.Snapshot.Remotes[name] = config.RemoteConfig{
		Type: typ,
		Path: path,
	}

	if err := SaveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Fprintf(w, "Remote '%s' added successfully.\n", name)
	return nil
}

func RunSnapshotRemoteList(engine *core.Engine, w io.Writer) error {
	// Engine config is sufficient for reading
	remotes := engine.GetConfig().Snapshot.Remotes
	if len(remotes) == 0 {
		fmt.Fprintln(w, "No remotes configured.")
		return nil
	}

	fmt.Fprintln(w, "NAME\tTYPE\tPATH")
	for name, r := range remotes {
		fmt.Fprintf(w, "%s\t%s\t%s\n", name, r.Type, r.Path)
	}
	return nil
}

func RunSnapshotRemoteRemove(engine *core.Engine, name string, w io.Writer) error {
	cfg, err := LoadConfigForEdit()
	if err != nil {
		return err
	}

	if _, ok := cfg.Snapshot.Remotes[name]; !ok {
		return fmt.Errorf("remote '%s' not found", name)
	}

	delete(cfg.Snapshot.Remotes, name)

	if err := SaveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Fprintf(w, "Remote '%s' removed.\n", name)
	return nil
}

func RunSnapshotRemoteTest(engine *core.Engine, name string, w io.Writer) error {
	// Stub: just check if it exists in config
	remotes := engine.GetConfig().Snapshot.Remotes
	if _, ok := remotes[name]; !ok {
		return fmt.Errorf("remote '%s' not found", name)
	}
	fmt.Fprintf(w, "Testing connection to '%s'...\n", name)
	// Real implementation would invoke rclone about/list
	fmt.Fprintln(w, "Connection successful (stub).")
	return nil
}
