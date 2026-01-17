package logger

import (
	"log/slog"
	"os"
)

// Init initializes the global logger.
// debug: enables LevelDebug
// verbose: enables LevelInfo (default is LevelWarn)
func Init(debug, verbose bool) {
	level := slog.LevelWarn // Default to silent-ish

	if debug {
		level = slog.LevelDebug
	} else if verbose {
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
		// AddSource: debug, // Optional: add source file handling for debug, maybe too verbose for CLI
	}

	// Use TextHandler for CLI friendliness, writing to Stderr to avoid mixing with Stdout
	handler := slog.NewTextHandler(os.Stderr, opts)
	logger := slog.New(handler)

	slog.SetDefault(logger)
}
