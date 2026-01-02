package output

import (
"fmt"
"os"
)

// ANSI color codes
const (
Reset   = "\033[0m"
Red     = "\033[31m"
Green   = "\033[32m"
Yellow  = "\033[33m"
Blue    = "\033[34m"
Magenta = "\033[35m"
Cyan    = "\033[36m"
White   = "\033[37m"
Bold    = "\033[1m"
)

var (
colorEnabled = true
)

// SetColorEnabled enables or disables colored output
func SetColorEnabled(enabled bool) {
	colorEnabled = enabled
}

// IsColorEnabled returns whether color output is enabled
func IsColorEnabled() bool {
	return colorEnabled
}

// InitColor initializes color settings based on flags and environment
func InitColor(forceColor, noColor bool) {
	if noColor {
		colorEnabled = false
		return
	}
	if forceColor {
		colorEnabled = true
		return
	}
	// Auto-detect: disable if not a terminal or NO_COLOR env is set
	if os.Getenv("NO_COLOR") != "" {
		colorEnabled = false
		return
	}
	// Check if stdout is a terminal
	fi, _ := os.Stdout.Stat()
	if (fi.Mode() & os.ModeCharDevice) == 0 {
		colorEnabled = false
	}
}

// Colorize wraps text with color codes if colors are enabled
func Colorize(color, text string) string {
	if !colorEnabled {
		return text
	}
	return color + text + Reset
}

// Success prints a green success message
func Success(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Println(Colorize(Green, "✓ "+msg))
}

// Error prints a red error message
func Error(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Fprintln(os.Stderr, Colorize(Red, "✗ "+msg))
}

// Warning prints a yellow warning message
func Warning(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Println(Colorize(Yellow, "⚠ "+msg))
}

// Info prints a blue info message
func Info(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Println(Colorize(Cyan, "→ "+msg))
}

// Bold prints bold text
func BoldText(text string) string {
	return Colorize(Bold, text)
}
