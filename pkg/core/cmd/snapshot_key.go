package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/pkg/core"
)

// SnapshotKeyCmd is the command to manage encryption keys
var SnapshotKeyCmd = &cobra.Command{
	Use:   "key",
	Short: "Manage snapshot encryption keys",
}

// Subcommands
var SnapshotKeyGenerateCmd = &cobra.Command{
	Use:   "generate <description>",
	Short: "Generate a new key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}
		return RunSnapshotKeyGenerate(engine, args, cmd.OutOrStdout())
	},
}

var SnapshotKeyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List keys",
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}
		return RunSnapshotKeyList(engine, cmd.OutOrStdout())
	},
}

var SnapshotKeyExportCmd = &cobra.Command{
	Use:   "export <id> [path]",
	Short: "Export a key",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}
		return RunSnapshotKeyExport(engine, args, cmd.OutOrStdout())
	},
}

var SnapshotKeyImportCmd = &cobra.Command{
	Use:   "import <path>",
	Short: "Import a key from file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}
		return RunSnapshotKeyImport(engine, args[0], cmd.OutOrStdout())
	},
}

var SnapshotKeyDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}
		return RunSnapshotKeyDelete(engine, args[0], cmd.OutOrStdout())
	},
}

func RunSnapshotKeyGenerate(engine *core.Engine, args []string, w io.Writer) error {
	km := engine.GetKeyManager()
	if km == nil {
		return fmt.Errorf("key manager not available")
	}

	desc := args[0]
	id, err := km.Generate(desc)
	if err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}

	_, _ = fmt.Fprintf(w, "Key generated successfully. ID: %s\n", id)

	return nil
}

func RunSnapshotKeyList(engine *core.Engine, w io.Writer) error {
	km := engine.GetKeyManager()
	if km == nil {
		return fmt.Errorf("key manager not available")
	}

	keys, err := km.List()
	if err != nil {
		return fmt.Errorf("failed to list keys: %w", err)
	}

	if len(keys) == 0 {
		_, _ = fmt.Fprintln(w, "No keys found.")

		return nil
	}

	for _, k := range keys {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", k.ID, k.Description)

	}
	return nil
}

func RunSnapshotKeyExport(engine *core.Engine, args []string, w io.Writer) error {
	km := engine.GetKeyManager()
	if km == nil {
		return fmt.Errorf("key manager not available")
	}

	id := args[0]
	if len(args) > 1 {
		if err := km.Export(id, args[1]); err != nil {
			return fmt.Errorf("failed to export key: %w", err)
		}
		_, _ = fmt.Fprintf(w, "Key exported to %s\n", args[1])
		return nil
	}

	tmp, err := os.CreateTemp("", "shedman-key-*.asc")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer func() { _ = os.Remove(tmpPath) }()

	if err := km.Export(id, tmpPath); err != nil {
		return fmt.Errorf("failed to export key: %w", err)
	}
	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return err
	}
	_, _ = w.Write(data)
	return nil
}

func RunSnapshotKeyImport(engine *core.Engine, path string, w io.Writer) error {
	km := engine.GetKeyManager()
	if km == nil {
		return fmt.Errorf("key manager not available")
	}
	if err := km.Import(path); err != nil {
		return fmt.Errorf("failed to import key: %w", err)
	}
	_, _ = fmt.Fprintf(w, "Key imported from %s\n", path)
	return nil
}

func RunSnapshotKeyDelete(engine *core.Engine, id string, w io.Writer) error {
	km := engine.GetKeyManager()
	if km == nil {
		return fmt.Errorf("key manager not available")
	}
	if err := km.Delete(id); err != nil {
		return fmt.Errorf("failed to delete key: %w", err)
	}
	_, _ = fmt.Fprintf(w, "Key deleted: %s\n", id)
	return nil
}

func init() {
	SnapshotKeyCmd.AddCommand(SnapshotKeyGenerateCmd)
	SnapshotKeyCmd.AddCommand(SnapshotKeyListCmd)
	SnapshotKeyCmd.AddCommand(SnapshotKeyExportCmd)
	SnapshotKeyCmd.AddCommand(SnapshotKeyImportCmd)
	SnapshotKeyCmd.AddCommand(SnapshotKeyDeleteCmd)
}
