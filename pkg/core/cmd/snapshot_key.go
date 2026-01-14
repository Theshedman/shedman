package cmd

import (
	"fmt"
	"io"

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

// Logic implementations

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

	fmt.Fprintf(w, "Key generated successfully. ID: %s\n", id)
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
		fmt.Fprintln(w, "No keys found.")
		return nil
	}

	for _, k := range keys {
		fmt.Fprintf(w, "%s\t%s\n", k.ID, k.Description)
	}
	return nil
}

func init() {
	SnapshotKeyCmd.AddCommand(SnapshotKeyGenerateCmd)
	SnapshotKeyCmd.AddCommand(SnapshotKeyListCmd)
	// Add others (Export, Import, Delete) similarly later
}
