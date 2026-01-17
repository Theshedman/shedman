package log

import (
	"fmt"
	"os"
	"time"

	"github.com/theshedman/shedman/internal/util"
)

// Logger handles transaction logging
type Logger struct {
	path string
}

// New creates a new logger
func New(path string) *Logger {
	return &Logger{
		path: path,
	}
}

// Transaction represents a package transaction
type Transaction struct {
	ID        string
	Timestamp time.Time
	Action    string
	Packages  []string
	Success   bool
}

// Log logs a transaction to the log file in ALPM format
func (l *Logger) Log(tx Transaction) error {
	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, util.FilePermissions)
	if err != nil {

		return err
	}
	defer f.Close()

	// Format: [YYYY-MM-DDTHH:MM:SS-0700] [ALPM] message

	timestamp := tx.Timestamp.Format("2006-01-02T15:04:05-0700")

	// Helper to write line
	writeLine := func(msg string) error {
		_, err := fmt.Fprintf(f, "[%s] [ALPM] %s\n", timestamp, msg)
		return err
	}

	if err := writeLine(fmt.Sprintf("transaction started")); err != nil {
		return err
	}

	for _, pkg := range tx.Packages {
		// Attempt to guess action from struct content or default to tx.Action
		if err := writeLine(fmt.Sprintf("%s %s", tx.Action, pkg)); err != nil {
			return err
		}
	}

	status := "completed"
	if !tx.Success {
		status = "failed"
	}

	return writeLine(fmt.Sprintf("transaction %s", status))
}

// List returns a list of transactions from the log file
func (l *Logger) List() ([]Transaction, error) {
	return Parse(l.path)
}
