package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/output"
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
	// Keep init separately if needed, or maybe just 'keyring init'
	KeyringCmd.AddCommand(newKeyringInitCmd())
}

func newKeyringListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List trusted keys",
		Run: func(cmd *cobra.Command, args []string) {
			eng := mustGetEngine()
			if err := RunKeyringList(eng); err != nil {
				output.Error("Failed to list keys: %v", err)
			}
		},
	}
}

func newKeyringAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add [keyid]",
		Short: "Add a key by ID",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			eng := mustGetEngine()
			if err := RunKeyringAdd(eng, args[0]); err != nil {
				output.Error("Failed to add key: %v", err)
			} else {
				output.Success("Key %s added.", args[0])
			}
		},
	}
}

func newKeyringRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove [keyid]",
		Short: "Remove a key by ID",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			eng := mustGetEngine()
			if err := RunKeyringRemove(eng, args[0]); err != nil {
				output.Error("Failed to remove key: %v", err)
			} else {
				output.Success("Key %s removed.", args[0])
			}
		},
	}
}

func newKeyringRefreshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh",
		Short: "Refresh keys from keyservers",
		Run: func(cmd *cobra.Command, args []string) {
			eng := mustGetEngine()
			if err := RunKeyringRefresh(eng); err != nil {
				output.Error("Failed to refresh keys: %v", err)
			} else {
				output.Success("Keys refreshed.")
			}
		},
	}
}

func newKeyringImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import [file]",
		Short: "Import key from file",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			eng := mustGetEngine()
			if err := RunKeyringImport(eng, args[0]); err != nil {
				output.Error("Failed to import key: %v", err)
			} else {
				output.Success("Key imported from %s.", args[0])
			}
		},
	}
}

func newKeyringInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize keyring",
		Run: func(cmd *cobra.Command, args []string) {
			eng := mustGetEngine()
			if err := RunKeyringInit(eng); err != nil {
				output.Error("Keyring init failed: %v", err)
			} else {
				output.Success("Keyring initialized.")
			}
		},
	}
}

// Helper to avoid repetition
func mustGetEngine() *core.Engine {
	eng, err := NewEngineWithConfig(nil)
	if err != nil {
		output.Error("Failed to initialize engine: %v", err)
		return nil // Should exit or panic in real app, but output.Error might not exit
	}
	return eng
}

// Logic Functions

func RunKeyringInit(eng *core.Engine) error {
	output.Info("Initializing keyring...")
	return eng.KeyringInit()
}

func RunKeyringRefresh(eng *core.Engine) error {
	output.Info("Refreshing keys...")
	return eng.KeyringRefresh()
}

func RunKeyringList(eng *core.Engine) error {
	keys, err := eng.KeyringList()
	if err != nil {
		return err
	}
	for _, k := range keys {
		fmt.Println(k)
	}
	return nil
}

func RunKeyringAdd(eng *core.Engine, keyID string) error {
	output.Info("Adding key %s...", keyID)
	return eng.KeyringAdd(keyID)
}

func RunKeyringRemove(eng *core.Engine, keyID string) error {
	output.Info("Removing key %s...", keyID)
	return eng.KeyringRemove(keyID)
}

func RunKeyringImport(eng *core.Engine, path string) error {
	output.Info("Importing key from %s...", path)
	return eng.KeyringImport(path)
}
