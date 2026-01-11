package log

import "time"

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

// Log logs a transaction
func (l *Logger) Log(tx Transaction) error {
	return nil
}
