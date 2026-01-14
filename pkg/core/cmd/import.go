package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/pkg/core"
)

var ImportCmd = &cobra.Command{
	Use:   "import [file]",
	Short: "Import packages from a list (file or stdin)",
	Long: `Import a list of packages from a file or standard input.
The input should contain one package name per line.
Empty lines and lines starting with '#' are ignored.

Usage:
  shedman import package_list.txt
  cat list.txt | shedman import -`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var reader io.Reader
		if len(args) == 0 || args[0] == "-" {
			reader = cmd.InOrStdin()
		} else {
			file, err := os.Open(args[0])
			if err != nil {
				return fmt.Errorf("failed to open file: %w", err)
			}
			defer file.Close()
			reader = file
		}

		eng, err := NewEngineWithConfig(nil)
		if err != nil {
			return err
		}
		return RunImport(eng, cmd.OutOrStdout(), reader)
	},
}

// RunImport imports packages from a file
func RunImport(eng *core.Engine, w io.Writer, r io.Reader) error {
	var pkgs []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		pkgs = append(pkgs, line)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}

	if len(pkgs) == 0 {
		fmt.Fprintln(w, "No packages found to import.")
		return nil
	}

	fmt.Fprintf(w, "Importing %d packages...\n", len(pkgs))

	// Use Install with Needed=true to avoid reinstalling existing
	opts := core.InstallOptions{
		Needed: true,
	}
	return eng.Install(pkgs, opts)
}
