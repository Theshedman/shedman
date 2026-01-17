package logger

import (
	"context"
	"log/slog"
	"testing"
)

func TestInit(t *testing.T) {
	// We can't easily capture os.Stderr in parallel tests without swapping it globally,
	// so we will test the logic by creating a custom handler manually or just trust Init sets the default.
	// For this unit test, we'll verify that SetDefault actually changes the default logger.

	// Reset default after test
	original := slog.Default()
	defer slog.SetDefault(original)

	Init(true, false) // Debug mode

	// Check if LevelDebug is enabled
	if !slog.Default().Enabled(context.TODO(), slog.LevelDebug) {

		t.Error("Expected LevelDebug to be enabled")
	}

	Init(false, true) // Verbose mode
	if slog.Default().Enabled(context.TODO(), slog.LevelDebug) {

		t.Error("Did not expect LevelDebug to be enabled in Verbose mode")
	}
	if !slog.Default().Enabled(context.TODO(), slog.LevelInfo) {

		t.Error("Expected LevelInfo to be enabled in Verbose mode")
	}

	Init(false, false) // Default mode
	if slog.Default().Enabled(context.TODO(), slog.LevelInfo) {

		t.Error("Did not expect LevelInfo to be enabled in Default mode")
	}
	if !slog.Default().Enabled(context.TODO(), slog.LevelWarn) {

		t.Error("Expected LevelWarn to be enabled in Default mode")
	}
}
