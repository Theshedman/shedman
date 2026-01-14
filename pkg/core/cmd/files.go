package cmd

import (
	"fmt"
	"io"

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
	RunE: func(cmd *cobra.Command, args []string) error {
		eng, err := NewEngineWithConfig(nil)
		if err != nil {
			return fmt.Errorf("failed to initialize engine: %w", err)
		}

		return RunFiles(eng, cmd.OutOrStdout(), args[0], filesSearch)
	},
}

func init() {
	FilesCmd.Flags().BoolVarP(&filesSearch, "search", "s", false, "Search file database for query")
}

func RunFiles(eng *core.Engine, w io.Writer, query string, search bool) error {
	if search {
		// Keeping output minimal or writing headers to w if needed.
		results, err := eng.SearchFiles(query)
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}
		if len(results) == 0 {
			output.Warning("File not found: %s", query)
			return nil
		}
		for _, line := range results {
			fmt.Fprintln(w, line)
		}
		return nil
	}

	// List files of a package
	files, err := eng.GetPackageFiles(query)
	if err != nil {
		return fmt.Errorf("failed to list files for %s: %w", query, err)
	}

	printFiles(w, query, files)
	return nil
}

func printFiles(w io.Writer, pkgName string, files []string) {
	fmt.Fprintf(w, "Files owned by %s (%d):\n", pkgName, len(files))
	for _, f := range files {
		fmt.Fprintf(w, "  %s\n", f)
	}
}
