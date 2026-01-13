package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
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

// RunFiles executes the files logic
func RunFiles(eng *core.Engine, w io.Writer, query string, search bool) error {
	if search {
		// output.Info writes to logger, we should acknowledge finding file in writer?
		// Keeping output minimal or writing headers to w if needed.
		// Tests verification doesn't enforce "Searching..." text, just results.

		results, err := eng.SearchFiles(query)
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}
		if len(results) == 0 {
			// output.Warning matches original, but maybe write to w?
			// To pass test "Expected 0 files" without error, we return nil.
			// Maybe write message.
			fmt.Fprintln(w, "No matching files found.")
			return nil
		}
		for _, line := range results {
			fmt.Fprintln(w, line)
		}
		return nil
	}

	// List files of a package
	// Iterate backends to find one that supports FileProvider and has the package
	// Or just any FileProvider? GetPackageFiles usually requires package to be installed or known.

	// Try official backend first
	if ob := eng.GetOfficialBackend(); ob != nil {
		if fp, ok := ob.(core.FileProvider); ok {
			files, err := fp.GetPackageFiles(query)
			if err == nil {
				printFiles(w, query, files)
				return nil
			}
			// If error is PackageNotFound, try other backends?
			// If generic error, return it?
		}
	}

	// Try to find ANY backend that supports this
	// But Engine doesn't expose ListBackends list directly?
	// Wait, we used ListBackends in Search and it failed.
	// Engine hides backends slice (private).
	// We should add GetPackageFiles to Engine if we want to support this properly.
	// OR: utilize that we are inside `pkg/core/cmd` and `Engine` struct fields are private to `package core`.
	// Use `core` package? `Engine` is in `pkg/core` (package core).
	// `files.go` is in `package cmd`. It CANNOT access `eng.backends`.

	// So we can ONLY use exposed methods.
	// Exposed: `GetOfficialBackend`, `SearchFiles`, `GetFileOwner`.
	// `GetPackageFiles` is NOT exposed on Engine.

	// Current `files.go` implementation only used OfficialBackend!
	// So we stick to that for now to satisfy existing logic, but handle type assertion safely.

	ob := eng.GetOfficialBackend()
	if ob == nil {
		return core.ErrBackendNotFound
	}

	fp, ok := ob.(core.FileProvider)
	if !ok {
		return fmt.Errorf("backend does not support file listing")
	}

	files, err := fp.GetPackageFiles(query)
	if err != nil {
		return fmt.Errorf("failed to list files for %s: %w", query, err)
	}

	printFiles(w, query, files)
	return nil
}

func printFiles(w io.Writer, pkgName string, files []string) {
	// Header usually printed.
	// Test checks for file presence.
	fmt.Fprintf(w, "Files owned by %s (%d):\n", pkgName, len(files))
	for _, f := range files {
		fmt.Fprintf(w, "  %s\n", f)
	}
}
