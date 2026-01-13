package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/output"
	"github.com/theshedman/shedman/pkg/core"
)

var filesSearch bool

// FilesCmd represents the files command
var FilesCmd = &cobra.Command{
	Use:   "files [package]",
	Short: "List files owned by a package or search for files",
	Long: `List all files owned by a specific package, or search the file database for a file.
Example:
  shedman files neovim           # List files of installed package
  shedman files --search vimrc   # Search for file in database`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("requires exactly one argument (package name or search query)")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		eng, err := NewEngineWithConfig(nil)
		if err != nil {
			output.Error("Failed to initialize engine: %v", err)
			return
		}

		if err := RunFiles(eng, args[0], filesSearch); err != nil {
			output.Error("%v", err)
		}
	},
}

func init() {
	FilesCmd.Flags().BoolVarP(&filesSearch, "search", "s", false, "Search file database for query")
}

// RunFiles executes the files logic
func RunFiles(eng *core.Engine, query string, search bool) error {
	if search {
		output.Info("Searching file database for '%s'...", query)
		results, err := eng.SearchFiles(query)
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}
		if len(results) == 0 {
			output.Warning("No matching files found.")
			return nil
		}
		for _, line := range results {
			fmt.Println(line)
		}
		return nil
	}

	// Default: List files of a package
	// Default: List files of a package
	// Use official backend directly as Engine wrapper doesn't expose package file listing yet

	backend := eng.GetOfficialBackend()
	if backend == nil {
		return core.ErrBackendNotFound
	}

	fp, ok := backend.(core.FileProvider)
	if !ok {
		return fmt.Errorf("backend does not support file listing")
	}

	files, err := fp.GetPackageFiles(query)
	if err != nil {
		return fmt.Errorf("failed to list files for %s: %w", query, err)
	}

	output.Info("Files owned by %s (%d):", query, len(files))
	for _, f := range files {
		fmt.Println("  " + f)
	}

	return nil
}
