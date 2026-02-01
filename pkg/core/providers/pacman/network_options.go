package pacman

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/theshedman/shedman/pkg/core"
)

const defaultDeltaRatio = "0.7"

func (b *Backend) preparePacmanArgs(baseArgs []string, opts core.UpgradeOptions) ([]string, func(), error) {
	args := append([]string{}, baseArgs...)
	cleanup := func() {}

	if opts.Delta {
		args = append(args, "--deltaratio", defaultDeltaRatio)
	}

	xferCommand := buildXferCommand(opts)
	if xferCommand == "" {
		return args, cleanup, nil
	}

	path, err := writePacmanConfig(b.configPath, xferCommand)
	if err != nil {
		return nil, nil, err
	}
	args = append(args, "--config", path)
	cleanup = func() {
		_ = os.Remove(path)
	}
	return args, cleanup, nil
}

func buildXferCommand(opts core.UpgradeOptions) string {
	if opts.LimitRate == "" && opts.Retry <= 0 && opts.Timeout <= 0 {
		return ""
	}

	parts := []string{"/usr/bin/curl", "-L", "-C", "-", "-f", "-o", "%o", "%u"}
	if opts.LimitRate != "" {
		parts = append(parts, "--limit-rate", opts.LimitRate)
	}
	if opts.Retry > 0 {
		parts = append(parts, "--retry", strconv.Itoa(opts.Retry))
	}
	if opts.Timeout > 0 {
		timeout := strconv.Itoa(opts.Timeout)
		parts = append(parts, "--connect-timeout", timeout, "--max-time", timeout)
	}
	return strings.Join(parts, " ")
}

func writePacmanConfig(basePath, xferCommand string) (string, error) {
	if xferCommand == "" {
		return "", fmt.Errorf("xfer command is required")
	}
	data, err := os.ReadFile(basePath)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(data), "\n")
	var out []string
	inOptions := false
	inserted := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			if inOptions && !inserted {
				out = append(out, "XferCommand = "+xferCommand)
				inserted = true
			}
			inOptions = strings.EqualFold(trimmed, "[options]")
			out = append(out, line)
			continue
		}

		if inOptions && strings.HasPrefix(strings.ToLower(trimmed), "xfercommand") {
			if !inserted {
				out = append(out, "XferCommand = "+xferCommand)
				inserted = true
			}
			continue
		}
		out = append(out, line)
	}

	if inOptions && !inserted {
		out = append(out, "XferCommand = "+xferCommand)
		inserted = true
	}

	if !inserted {
		out = append([]string{"[options]", "XferCommand = " + xferCommand, ""}, out...)
	}

	tmp, err := os.CreateTemp("", "pacman-conf-*.conf")
	if err != nil {
		return "", err
	}
	if _, err := tmp.WriteString(strings.Join(out, "\n")); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return "", err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return "", err
	}
	return tmp.Name(), nil
}
