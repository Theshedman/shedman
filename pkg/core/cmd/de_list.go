package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/pkg/de"
)

var deListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available desktop environments",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadDefault()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
			os.Exit(1)
		}

		engine, err := NewEngineWithConfig(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to initialize engine: %v\n", err)
			os.Exit(1)
		}

		mgr := de.New(engine)
		des, err := mgr.List()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to list DEs: %v\n", err)
			os.Exit(1)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tINSTALLED\tPACKAGE")
		for _, d := range des {
			installed := ""
			if d.Installed {
				installed = "[installed]"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", d.ID, d.Name, installed, d.Package)
		}
		w.Flush()
	},
}
