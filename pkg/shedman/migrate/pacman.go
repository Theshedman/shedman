package migrate

import (
"bufio"
"os"
"strconv"
"strings"
)

// PacmanConfig holds parsed pacman.conf settings
type PacmanConfig struct {
	ParallelDownloads int
	IgnorePkg         []string
	IgnoreGroup       []string
	HoldPkg           []string
	SigLevel          string
	Mirrors           []string
}

// ParsePacmanConf parses a pacman.conf file and extracts relevant settings
func ParsePacmanConf(path string) (*PacmanConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := &PacmanConfig{
		ParallelDownloads: 1, // default
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key = value pairs
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}

			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			switch key {
			case "ParallelDownloads":
				if n, err := strconv.Atoi(value); err == nil {
					config.ParallelDownloads = n
				}
			case "IgnorePkg":
				config.IgnorePkg = parseSpaceSeparated(value)
			case "IgnoreGroup":
				config.IgnoreGroup = parseSpaceSeparated(value)
			case "HoldPkg":
				config.HoldPkg = parseSpaceSeparated(value)
			case "SigLevel":
				config.SigLevel = value
			case "Server":
				config.Mirrors = append(config.Mirrors, value)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return config, nil
}

// parseSpaceSeparated splits a space-separated string into a slice
func parseSpaceSeparated(s string) []string {
	var result []string
	for _, item := range strings.Fields(s) {
		result = append(result, item)
	}
	return result
}
