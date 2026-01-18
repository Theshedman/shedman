package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/internal/output"
	"github.com/theshedman/shedman/internal/util"
	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/executor"
	"github.com/theshedman/shedman/pkg/snapshot"
	"github.com/theshedman/shedman/pkg/snapshot/restic"
)

var SnapshotRemoteCmd = &cobra.Command{
	Use:   "remote",
	Short: "Manage remote snapshots",
}

var SnapshotRemoteInitCmd = &cobra.Command{
	Use:   "init <remote-name>",
	Short: "Initialize a restic repository on the remote",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}
		return RunSnapshotRemoteInit(engine, args[0], cmd.OutOrStdout())
	},
}

var SnapshotRemotePushCmd = &cobra.Command{
	Use:   "push <id> [target-name] [path]",
	Short: "Push a snapshot to a remote target",
	Args:  cobra.RangeArgs(1, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}

		targetName := ""
		if len(args) > 1 {
			targetName = args[1]
		}

		path := ""
		if len(args) > 2 {
			path = args[2]
		}

		cfg := engine.GetConfig()
		if targetName == "" {
			if cfg.Snapshot.DefaultRemote != "" {
				targetName = cfg.Snapshot.DefaultRemote
			} else {
				if len(cfg.Snapshot.Remotes) == 1 {
					for k := range cfg.Snapshot.Remotes {
						targetName = k
						break
					}
				} else {
					return fmt.Errorf("no target specified and no default_remote configured (remotes available: %d)", len(cfg.Snapshot.Remotes))
				}
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Using default remote: %s\n", targetName)

		}

		target := snapshot.RemoteTarget{
			Name: targetName,
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
	Use:   "pull <id> [source-name] [path]",
	Short: "Pull a snapshot from a remote source",
	Args:  cobra.RangeArgs(1, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}

		sourceName := ""
		if len(args) > 1 {
			sourceName = args[1]
		}

		path := ""
		if len(args) > 2 {
			path = args[2]
		}

		cfg := engine.GetConfig()
		if sourceName == "" {
			if cfg.Snapshot.DefaultRemote != "" {
				sourceName = cfg.Snapshot.DefaultRemote
			} else {
				if len(cfg.Snapshot.Remotes) == 1 {
					for k := range cfg.Snapshot.Remotes {
						sourceName = k
						break
					}
				} else {
					return fmt.Errorf("no source specified and no default_remote configured")
				}
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Using default remote: %s\n", sourceName)

		}

		source := snapshot.RemoteTarget{
			Name: sourceName,
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
	Use:   "add <rclone-remote-name>",
	Short: "Add an existing rclone remote",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}
		return RunSnapshotRemoteAdd(engine, args[0], cmd.OutOrStdout())
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
	SnapshotRemoteCmd.AddCommand(SnapshotRemoteInitCmd)
	SnapshotRemoteCmd.AddCommand(SnapshotRemotePushCmd)
	SnapshotRemoteCmd.AddCommand(SnapshotRemotePullCmd)
	SnapshotRemoteCmd.AddCommand(SnapshotRemoteAddCmd)
	SnapshotRemoteCmd.AddCommand(SnapshotRemoteListCmd)
	SnapshotRemoteCmd.AddCommand(SnapshotRemoteRemoveCmd)
	SnapshotRemoteCmd.AddCommand(SnapshotRemoteTestCmd)

	SnapshotRemotePushCmd.Flags().BoolVar(&snapshotRemoteDelete, "delete", false, "Delete extraneous files on destination")
	SnapshotRemotePushCmd.Flags().IntVar(&snapshotRemoteBandwidth, "bwlimit", 0, "Bandwidth limit in KB/s")

	SnapshotRemotePullCmd.Flags().BoolVar(&snapshotRemoteDelete, "delete", false, "Delete extraneous files on local (cautious usage)")
	SnapshotRemotePullCmd.Flags().IntVar(&snapshotRemoteBandwidth, "bwlimit", 0, "Bandwidth limit in KB/s")
}

var (
	snapshotRemoteCompress  bool
	snapshotRemoteDelete    bool
	snapshotRemoteBandwidth int
)

func RunSnapshotRemoteAdd(engine *core.Engine, name string, w io.Writer) error {
	out, err := (&executor.RealExecutor{}).Output("rclone", "listremotes")
	if err != nil {

		return fmt.Errorf("failed to list rclone remotes (is rclone installed?): %w", err)
	}

	remotes := strings.Split(string(out), "\n")
	found := false
	for _, r := range remotes {
		if strings.TrimSpace(r) == name+":" {
			found = true
			break
		}
	}

	if !found {
		_, _ = fmt.Fprintf(w, "Remote '%s' not found locally.\n", name)

		create, err := output.ReadInput("Do you want to configure it via rclone? [Y/n]: ")
		if err != nil {
			return err
		}
		if create == "" || strings.ToLower(create) == "y" || strings.ToLower(create) == "yes" {
			typ, err := output.ReadInput("Enter remote type (e.g. drive, s3) [default: drive]: ")
			if err != nil {
				return err
			}
			if typ == "" {
				typ = "drive"
			}

			_, _ = fmt.Fprintf(w, "Launching rclone config for '%s' (%s).\n", name, typ)

			var args []string
			args = append(args, "config", "create", name, typ)

			switch strings.ToLower(typ) {
			case "gdrive", "drive":
			case "s3":
				accessKey := util.GetEnvOrPrompt("AWS_ACCESS_KEY_ID", "Enter AWS Access Key ID: ")
				secretKey := util.GetEnvOrPrompt("AWS_SECRET_ACCESS_KEY", "Enter AWS Secret Access Key: ")
				region := util.GetEnvOrPrompt("AWS_REGION", "Enter Region (e.g. us-east-1): ")
				endpoint, _ := output.ReadInput("Enter Endpoint (optional, leave blank for AWS): ")

				args = append(args, "env_auth=false")
				args = append(args, "access_key_id="+accessKey)
				args = append(args, "secret_access_key="+secretKey)
				if region != "" {
					args = append(args, "region="+region)
				}
				if endpoint != "" {
					args = append(args, "endpoint="+endpoint)
				}

			case "r2":
				args[3] = "s3"
				args = append(args, "provider=Cloudflare")

				accessKey := util.GetEnvOrPrompt("R2_ACCESS_KEY_ID", "Enter R2 Access Key ID: ")
				secretKey := util.GetEnvOrPrompt("R2_SECRET_ACCESS_KEY", "Enter R2 Secret Access Key: ")
				accountID := util.GetEnvOrPrompt("R2_ACCOUNT_ID", "Enter Cloudflare Account ID: ")

				endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)

				args = append(args, "access_key_id="+accessKey)
				args = append(args, "secret_access_key="+secretKey)
				args = append(args, "endpoint="+endpoint)

			default:
			}

			cmd := (&executor.RealExecutor{}).Command("rclone", args...)
			cmd.Stdin = os.Stdin
			cmd.Stdout = w

			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				return fmt.Errorf("rclone config failed: %w", err)
			}
			_, _ = fmt.Fprintln(w, "\n----")
			_, _ = fmt.Fprintln(w, "Rclone configuration completed.")

		} else {
			return fmt.Errorf("remote '%s' not found. Please configure it via 'rclone config' first.\nAvailable remotes: %s", name, strings.Join(remotes, ", "))
		}
	}

	cfg, err := config.LoadDefault()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.Snapshot.Remotes == nil {
		cfg.Snapshot.Remotes = make(map[string]config.RemoteConfig)
	}

	if cfg.Snapshot.Remotes == nil {
		cfg.Snapshot.Remotes = make(map[string]config.RemoteConfig)
	}

	cfg.Snapshot.Remotes[name] = config.RemoteConfig{
		Type: "rclone",
		Path: name + ":",
	}

	if cfg.Snapshot.DefaultRemote == "" {
		cfg.Snapshot.DefaultRemote = name
		_, _ = fmt.Fprintf(w, "Marked '%s' as default remote.\n", name)

	}

	if err := config.Save(config.DefaultConfigPath(), cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	_, _ = fmt.Fprintf(w, "Remote '%s' added successfully to shedman.\n", name)

	return nil
}

func RunSnapshotRemoteList(engine *core.Engine, w io.Writer) error {
	remotes := engine.GetConfig().Snapshot.Remotes
	if len(remotes) == 0 {
		_, _ = fmt.Fprintln(w, "No remotes configured.")

		return nil
	}

	_, _ = fmt.Fprintln(w, "NAME\tTYPE\tPATH")
	for name, r := range remotes {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", name, r.Type, r.Path)
	}

	return nil
}

func RunSnapshotRemoteRemove(engine *core.Engine, name string, w io.Writer) error {
	cfg, err := config.LoadDefault()
	if err != nil {
		return err
	}

	if _, ok := cfg.Snapshot.Remotes[name]; !ok {
		return fmt.Errorf("remote '%s' not found", name)
	}

	delete(cfg.Snapshot.Remotes, name)

	if err := config.Save(config.DefaultConfigPath(), cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	_, _ = fmt.Fprintf(w, "Remote '%s' removed.\n", name)

	return nil
}

func RunSnapshotRemoteTest(engine *core.Engine, name string, w io.Writer) error {
	cfg := engine.GetConfig()
	remote, ok := cfg.Snapshot.Remotes[name]
	if !ok {
		return fmt.Errorf("remote '%s' not found in config", name)
	}

	_, _ = fmt.Fprintf(w, "Testing connectivity to '%s' (rclone)...\n", name)

	target := remote.Path
	if target == "" {
		target = name + ":"
	}

	out, err := (&executor.RealExecutor{}).Output("rclone", "about", target)
	if err != nil {

		return fmt.Errorf("connection failed: %w\nOutput: %s", err, string(out))
	}
	_, _ = fmt.Fprintln(w, "Success: Remote is accessible.")

	return nil
}

func RunSnapshotRemotePush(engine *core.Engine, id string, target snapshot.RemoteTarget, opts snapshot.RemoteOptions, w io.Writer) error {
	mgr := engine.GetSnapshotManager()
	if mgr == nil {
		return fmt.Errorf("snapshot manager not available")
	}

	// Lookup config
	if cfg := engine.GetConfig(); cfg != nil {
		if r, ok := cfg.Snapshot.Remotes[target.Name]; ok {
			target.Type = r.Type
			target.Path = r.Path
			// If not found in config, treat as ad-hoc remote (below)
		}
	}

	if target.Type == "" {
		target.Type = "rclone"
		if target.Path == "" {
			target.Path = target.Name + ":"
		}
	}

	_, _ = fmt.Fprintf(w, "Pushing snapshot %s to %s...\n", id, target.Path)
	if err := mgr.Push(id, target, opts); err != nil {
		return fmt.Errorf("push failed: %w", err)
	}
	_, _ = fmt.Fprintln(w, "Push successful.")

	return nil
}

func RunSnapshotRemotePull(engine *core.Engine, id string, source snapshot.RemoteTarget, opts snapshot.RemoteOptions, w io.Writer) error {
	mgr := engine.GetSnapshotManager()
	if mgr == nil {
		return fmt.Errorf("snapshot manager not available")
	}

	// Lookup config
	if cfg := engine.GetConfig(); cfg != nil {
		if r, ok := cfg.Snapshot.Remotes[source.Name]; ok {
			source.Type = r.Type
			source.Path = r.Path
		}
	}

	if source.Type == "" {
		source.Type = "rclone"
		if source.Path == "" {
			source.Path = source.Name + ":"
		}
	}

	_, _ = fmt.Fprintf(w, "Pulling snapshot %s from %s...\n", id, source.Path)
	if err := mgr.Pull(id, source, opts); err != nil {
		return fmt.Errorf("pull failed: %w", err)
	}
	_, _ = fmt.Fprintln(w, "Pull successful.")

	return nil
}

func RunSnapshotRemoteInit(engine *core.Engine, name string, w io.Writer) error {
	cfg := engine.GetConfig()
	strategy := cfg.Snapshot.RemoteStrategy

	if strategy != "restic" {
		_, _ = fmt.Fprintln(w, "Note: Initialization is only required for 'restic' strategy.")
		_, _ = fmt.Fprintf(w, "Current strategy is '%s' (default: rclone). Rclone remotes are standard folders.\n", strategy)
		return nil
	}

	remote, ok := cfg.Snapshot.Remotes[name]
	if !ok {
		return fmt.Errorf("remote '%s' not found in shedman config (run 'shedman snapshot remote add %s' first)", name, name)
	}

	remotePath := remote.Path
	if remotePath == "" {
		remotePath = name + ":"
	}

	pwd := util.GetEnvOrPrompt("RESTIC_PASSWORD", "Enter New Restic Repository Password: ")
	if pwd == "" {
		return fmt.Errorf("password is required to initialize restic repository")
	}

	// Create manager
	exec := &executor.RealExecutor{}
	mgr := restic.NewManager(exec, pwd)

	_, _ = fmt.Fprintf(w, "Initializing restic repository at %s...\n", remotePath)
	if err := mgr.Init(remotePath); err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}

	_, _ = fmt.Fprintln(w, "Repository initialized successfully! Password checks out.")
	return nil
}
