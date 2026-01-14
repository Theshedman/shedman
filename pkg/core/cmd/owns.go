package cmd

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/pkg/core"
)

// OwnsCmd represents the owns command
var OwnsCmd = &cobra.Command{
	Use:   "owns [file]",
	Short: "Check which package owns a file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := filepath.Abs(args[0])
		if err != nil {
			return fmt.Errorf("invalid path: %w", err)
		}

		eng, err := NewEngineWithConfig(nil)
		if err != nil {
			return fmt.Errorf("failed to initialize engine: %w", err)
		}

		return RunOwns(eng, cmd.OutOrStdout(), path)
	},
}

func RunOwns(eng *core.Engine, w io.Writer, path string) error {
	owner, err := eng.GetFileOwner(path)
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "File '%s' is owned by '%s'\n", path, owner)
	return nil
}
