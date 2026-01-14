package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/output"
)

var (
	historyLimit int
	historyJSON  bool
	historySince string
	historyPkg   string
)

// HistoryOptions holds options for history command
type HistoryOptions struct {
	Limit   int
	JSON    bool
	Since   time.Time
	Package string
}

// Transaction represents a parsed log entry
type Transaction struct {
	Date    time.Time `json:"date"`
	Action  string    `json:"action"`
	Package string    `json:"package"`
	Version string    `json:"version"`
}

// HistoryCmd represents the history command
var HistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "View package transaction history",
	Long:  `View the recent package transactions from the pacman log.`,
	Run: func(cmd *cobra.Command, args []string) {
		logPath := "/var/log/pacman.log"
		file, err := os.Open(logPath)
		if err != nil {
			output.Error("Failed to open log: %v", err)
			return
		}
		defer file.Close()

		var sinceTime time.Time
		if historySince != "" {
			// Try parsing date. Common formats: YYYY-MM-DD
			parsed, err := time.Parse("2006-01-02", historySince)
			if err != nil {
				output.Error("Invalid date format (use YYYY-MM-DD): %v", err)
				return
			}
			sinceTime = parsed
		}

		opts := HistoryOptions{
			Limit:   historyLimit,
			JSON:    historyJSON,
			Since:   sinceTime,
			Package: historyPkg,
		}

		if err := RunHistory(file, os.Stdout, opts); err != nil {
			output.Error("Failed to read history: %v", err)
		}
	},
}

func RunHistory(r io.Reader, w io.Writer, opts HistoryOptions) error {
	var transactions []Transaction
	scanner := bufio.NewScanner(r)

	// Log format: [2023-05-20T19:33:04-0400] [ALPM] installed foo (1.0-1)

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.Contains(line, "[ALPM]") {
			continue
		}

		endBracket := strings.Index(line, "]")
		if endBracket < 2 {
			continue
		}
		dateStr := line[1:endBracket]
		t, err := time.Parse("2006-01-02T15:04:05-0700", dateStr)
		if err != nil {
			continue
		}

		if !opts.Since.IsZero() && t.Before(opts.Since) {
			continue
		}

		alpmIdx := strings.Index(line, "[ALPM]")
		if alpmIdx == -1 {
			continue
		}

		rest := strings.TrimSpace(line[alpmIdx+6:])
		parts := strings.Fields(rest)
		if len(parts) < 3 {
			continue
		}

		action := parts[0] // installed, upgraded, removed
		pkgName := parts[1]
		version := parts[2]

		if action != "installed" && action != "upgraded" && action != "removed" && action != "reinstalled" {
			continue
		}

		if opts.Package != "" && pkgName != opts.Package {
			continue
		}

		version = strings.Trim(strings.Join(parts[2:], " "), "()")

		transactions = append(transactions, Transaction{
			Date:    t,
			Action:  action,
			Package: pkgName,
			Version: version,
		})
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	var displayed []Transaction
	count := 0
	for i := len(transactions) - 1; i >= 0; i-- {
		displayed = append(displayed, transactions[i])
		count++
		if opts.Limit > 0 && count >= opts.Limit {
			break
		}
	}

	if opts.JSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(displayed)
	}

	for _, tx := range displayed {
		fmt.Fprintf(w, "[%s] %s %s %s\n", tx.Date.Format("2006-01-02 15:04"), tx.Action, tx.Package, tx.Version)
	}

	return nil
}

func init() {
	HistoryCmd.Flags().IntVarP(&historyLimit, "limit", "n", 20, "Number of lines to show")
	HistoryCmd.Flags().BoolVar(&historyJSON, "json", false, "Output in JSON format")
	HistoryCmd.Flags().StringVar(&historySince, "since", "", "Show logs since date (YYYY-MM-DD)")
	HistoryCmd.Flags().StringVar(&historyPkg, "package", "", "Filter by package name")
}
