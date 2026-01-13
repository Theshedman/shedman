package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/pkg/core"
)

// KeyringCmd represents the keyring command
var KeyringCmd = &cobra.Command{
	Use:   "keyring",
	Short: "Manage GPG keys",
	Long:  `Manage pacman GPG keys: list, add, remove, refresh, and import.`,
}

func init() {
	KeyringCmd.AddCommand(newKeyringListCmd())
	KeyringCmd.AddCommand(newKeyringAddCmd())
	KeyringCmd.AddCommand(newKeyringRemoveCmd())
	KeyringCmd.AddCommand(newKeyringRefreshCmd())
	KeyringCmd.AddCommand(newKeyringImportCmd())
	KeyringCmd.AddCommand(newKeyringInitCmd())
}

func newKeyringListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List trusted keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			eng, err := NewEngineWithConfig(nil)
			if err != nil {
				return fmt.Errorf("failed to initialize engine: %w", err)
			}
			return RunKeyringList(eng, cmd.OutOrStdout())
		},
	}
}

func newKeyringAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add [keyid]",
		Short: "Add a key by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			eng, err := NewEngineWithConfig(nil)
			if err != nil {
				return fmt.Errorf("failed to initialize engine: %w", err)
			}
			if err := RunKeyringAdd(eng, cmd.OutOrStdout(), args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Key %s added.\n", args[0])
			return nil
		},
	}
}

func newKeyringRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove [keyid]",
		Short: "Remove a key by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			eng, err := NewEngineWithConfig(nil)
			if err != nil {
				return fmt.Errorf("failed to initialize engine: %w", err)
			}
			if err := RunKeyringRemove(eng, cmd.OutOrStdout(), args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Key %s removed.\n", args[0])
			return nil
		},
	}
}

func newKeyringRefreshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh",
		Short: "Refresh keys from keyservers",
		RunE: func(cmd *cobra.Command, args []string) error {
			eng, err := NewEngineWithConfig(nil)
			if err != nil {
				return fmt.Errorf("failed to initialize engine: %w", err)
			}
			if err := RunKeyringRefresh(eng, cmd.OutOrStdout()); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Keys refreshed.")
			return nil
		},
	}
}

func newKeyringImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import [file]",
		Short: "Import key from file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			eng, err := NewEngineWithConfig(nil)
			if err != nil {
				return fmt.Errorf("failed to initialize engine: %w", err)
			}
			if err := RunKeyringImport(eng, cmd.OutOrStdout(), args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Key imported from %s.\n", args[0])
			return nil
		},
	}
}

func newKeyringInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize keyring",
		RunE: func(cmd *cobra.Command, args []string) error {
			eng, err := NewEngineWithConfig(nil)
			if err != nil {
				return fmt.Errorf("failed to initialize engine: %w", err)
			}
			if err := RunKeyringInit(eng, cmd.OutOrStdout()); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Keyring initialized.")
			return nil
		},
	}
}

// Logic Functions

func RunKeyringInit(eng *core.Engine, w io.Writer) error {
	fmt.Fprintln(w, "Initializing keyring...")
	return eng.KeyringInit()
}

func RunKeyringRefresh(eng *core.Engine, w io.Writer) error {
	fmt.Fprintln(w, "Refreshing keys...")
	return eng.KeyringRefresh()
}

func RunKeyringList(eng *core.Engine, w io.Writer) error {
	keys, err := eng.KeyringList()
	if err != nil {
		return err
	}
	for _, k := range keys {
		fmt.Fprintln(w, k)
	}
	return nil
}

func RunKeyringAdd(eng *core.Engine, w io.Writer, keyID string) error {
	fmt.Fprintf(w, "Adding key %s...\n", keyID)
	return eng.KeyringAdd(keyID)
}

func RunKeyringRemove(eng *core.Engine, w io.Writer, keyID string) error {
	fmt.Fprintf(w, "Removing key %s...\n", keyID)
	return eng.KeyringRemove(keyID)
}

func RunKeyringImport(eng *core.Engine, w io.Writer, path string) error {
	fmt.Fprintf(w, "Importing key from %s...\n", path)
	return eng.KeyringImport(path)
}
