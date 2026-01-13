package pacman

import (
	"bufio"
	"strconv"
	"strings"
)

func parsePacmanSize(output, key string) int64 {
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(line, ":"); idx > 0 {
			k := strings.TrimSpace(line[:idx])
			if k == key {
				val := strings.TrimSpace(line[idx+1:])
				return parseSizeString(val)
			}
		}
	}
	return 0
}

func parseSizeString(s string) int64 {
	parts := strings.Fields(s)
	if len(parts) < 2 {
		return 0
	}
	val, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0
	}
	unit := parts[1]

	multiplier := int64(1)
	switch unit {
	case "KiB":
		multiplier = 1024
	case "MiB":
		multiplier = 1024 * 1024
	case "GiB":
		multiplier = 1024 * 1024 * 1024
	}
	return int64(val * float64(multiplier))
}
