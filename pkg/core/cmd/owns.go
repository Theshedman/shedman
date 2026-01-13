package cmd

import (
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/output"
	"github.com/theshedman/shedman/pkg/core"
)

// OwnsCmd represents the owns command
var OwnsCmd = &cobra.Command{
	Use:   "owns [file]",
	Short: "Check which package owns a file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path, err := filepath.Abs(args[0])
		if err != nil {
			output.Error("Invalid path: %v", err)
			return
		}

		eng, err := NewEngineWithConfig(nil)
		if err != nil {
			output.Error("Failed to initialize engine: %v", err)
			return
		}

		if err := RunOwns(eng, path); err != nil {
			output.Error("%v", err)
			return
		}
	},
}

// RunOwns executes the owns logic
func RunOwns(eng *core.Engine, path string) error {
	owner, err := eng.GetFileOwner(path)
	if err != nil {
		return err
	}

	output.Success("File '%s' is owned by '%s'", path, owner)
	return nil
}
