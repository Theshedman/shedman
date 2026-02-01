package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/pkg/log"
)

var (
	logJSON  bool
	logSince string
	logLimit int
)

// LogCmd represents the log command.
var LogCmd = &cobra.Command{
	Use:   "log",
	Short: "Show recent shedman logs",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefault()
		if err != nil {
			return err
		}

		path := cfg.Logging.File
		if path == "" {
			path = "/var/log/shedman/shedman.log"
		}
		if _, err := os.Stat(path); err != nil && os.IsNotExist(err) {
			path = "/var/log/pacman.log"
		}

		txs, err := log.Parse(path)
		if err != nil {
			return err
		}

		var since time.Time
		if logSince != "" {
			dur, err := parseDurationOrDays(logSince)
			if err != nil {
				return err
			}
			since = time.Now().Add(-dur)
		}

		var filtered []log.Transaction
		for _, tx := range txs {
			if !since.IsZero() && tx.Timestamp.Before(since) {
				continue
			}
			filtered = append(filtered, tx)
		}

		if logLimit > 0 && len(filtered) > logLimit {
			filtered = filtered[len(filtered)-logLimit:]
		}

		if logJSON {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(filtered)
		}

		for _, tx := range filtered {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s %v\n",
				tx.Timestamp.Format("2006-01-02 15:04"),
				tx.Action,
				tx.Packages)
		}
		return nil
	},
}

func init() {
	LogCmd.Flags().BoolVar(&logJSON, "json", false, "Output in JSON format")
	LogCmd.Flags().StringVar(&logSince, "since", "", "Show logs since duration (e.g. 7d, 24h)")
	LogCmd.Flags().IntVar(&logLimit, "limit", 50, "Number of entries to show")
}

func parseDurationOrDays(val string) (time.Duration, error) {
	if len(val) > 1 && val[len(val)-1] == 'd' {
		var days int
		if _, err := fmt.Sscanf(val, "%dd", &days); err != nil {
			return 0, fmt.Errorf("invalid day duration: %s", val)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return 0, fmt.Errorf("invalid duration: %s", val)
	}
	return d, nil
}
