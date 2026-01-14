package util

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseSize parses human-readable file size strings (e.g. "10 MiB", "7.5 GB") into bytes.
// It supports B, KiB, MiB, GiB, TiB, KB, MB, GB, TB.
// It also parses simple raw integer strings (e.g. "1024").
func ParseSize(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" {
		return 0
	}

	// Try parsing as simple integer first (raw bytes)
	if val, err := strconv.ParseInt(s, 10, 64); err == nil {
		return val
	}

	// Split value and unit (handle multiple spaces)
	parts := strings.Fields(s)
	if len(parts) != 2 {
		return 0
	}

	valStr := parts[0]
	unit := parts[1]

	// Handle commas as decimals if locale issues exist (e.g. "7,50")
	valStr = strings.ReplaceAll(valStr, ",", ".")

	var val float64
	if _, err := fmt.Sscanf(valStr, "%f", &val); err != nil {
		return 0
	}

	var multiplier int64 = 1
	switch unit {
	case "B", "bytes":
		multiplier = 1
	case "KiB", "KB":
		multiplier = 1024
	case "MiB", "MB":
		multiplier = 1024 * 1024
	case "GiB", "GB":
		multiplier = 1024 * 1024 * 1024
	case "TiB", "TB":
		multiplier = 1024 * 1024 * 1024 * 1024
	case "PiB", "PB":
		multiplier = 1024 * 1024 * 1024 * 1024 * 1024
	}

	return int64(val * float64(multiplier))
}

// FormatSize formats bytes into human-readable string (IEC standard)
func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
