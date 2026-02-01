package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/pkg/mirror"
)

var (
	mirrorSelectTop     int
	mirrorSelectSort    string
	mirrorSelectCountry []string
	mirrorTestSelect    bool
)

// MirrorCmd represents mirror management.
var MirrorCmd = &cobra.Command{
	Use:   "mirror",
	Short: "Manage mirrors",
	Long:  "List, test, and select fastest mirrors.",
}

var mirrorListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured mirrors",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := mirror.New()
		mirrors, err := mgr.List()
		if err != nil {
			return err
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
		_, _ = fmt.Fprintln(w, "URL\tCOUNTRY")
		for _, m := range mirrors {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", m.URL, m.Country)
		}
		_ = w.Flush()
		return nil
	},
}

var mirrorTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test mirror speeds",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := mirror.New()
		results, err := mgr.Test()
		if err != nil {
			return err
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
		_, _ = fmt.Fprintln(w, "URL\tCOUNTRY\tLATENCY")
		for _, m := range results {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", m.URL, m.Country, m.Speed)
		}
		_ = w.Flush()

		if mirrorTestSelect {
			return mgr.Select(mirrorSelectTop, mirrorSelectCountry, mirrorSelectSort)
		}
		return nil
	},
}

var mirrorSelectCmd = &cobra.Command{
	Use:   "select",
	Short: "Select fastest mirrors",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := mirror.New()
		return mgr.Select(mirrorSelectTop, mirrorSelectCountry, mirrorSelectSort)
	},
}

func init() {
	MirrorCmd.AddCommand(mirrorListCmd)
	MirrorCmd.AddCommand(mirrorTestCmd)
	MirrorCmd.AddCommand(mirrorSelectCmd)

	mirrorTestCmd.Flags().BoolVar(&mirrorTestSelect, "select", false, "Auto-select fastest mirrors after testing")

	mirrorSelectCmd.Flags().IntVar(&mirrorSelectTop, "top", 5, "Number of mirrors to keep")
	mirrorSelectCmd.Flags().StringSliceVar(&mirrorSelectCountry, "country", nil, "Filter by country (comma-separated)")
	mirrorSelectCmd.Flags().StringVar(&mirrorSelectSort, "sort", "rate", "Sort method for reflector (rate, age, delay)")

	// Mirror test uses same selection flags when --select is provided
	mirrorTestCmd.Flags().IntVar(&mirrorSelectTop, "top", 5, "Number of mirrors to keep")
	mirrorTestCmd.Flags().StringSliceVar(&mirrorSelectCountry, "country", nil, "Filter by country (comma-separated)")
	mirrorTestCmd.Flags().StringVar(&mirrorSelectSort, "sort", "rate", "Sort method for reflector (rate, age, delay)")
}
